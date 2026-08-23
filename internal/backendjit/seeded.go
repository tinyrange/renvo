//go:build !renvo

package backendjit

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"sync"

	"renvo.dev/internal/backendbuiltin"
	"renvo.dev/internal/backendvm32"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgb"
	"renvo.dev/internal/unit"
	"renvo.dev/std/vm"
)

const (
	seededKernelVersion       = 3
	seededHostEmitterKernel   = 4
	seededCompilerArenaSize   = 1 << 30
	seededVMCompilerArenaSize = 32 << 20
	seededVMMemoryLimit       = 64 << 20
	seededVMStepLimit         = 20 * 1000 * 1000 * 1000
)

// SeededBackend derives a single-target compiler from the fixed VM32 seed.
// Built-in mode resolves the checked-in definition catalog; external mode
// resolves the definition at path.
type SeededBackend struct {
	prepareMu   sync.Mutex
	path        string
	stdRoot     string
	cacheDir    string
	builtin     bool
	prepared    Prepared
	preparedFor string
	vmPrepared  Prepared
	vmFor       string
}

func NewSeeded(path string, stdRoot string, cacheDir string) *SeededBackend {
	return &SeededBackend{path: path, stdRoot: stdRoot, cacheDir: cacheDir}
}

func NewBuiltin(stdRoot string, cacheDir string) *SeededBackend {
	return &SeededBackend{stdRoot: stdRoot, cacheDir: cacheDir, builtin: true}
}

// Prepare derives or loads the versioned compiler artifact for target.
func (backend *SeededBackend) Prepare(target string) Prepared {
	return backend.prepare(target)
}

func (backend *SeededBackend) CompileUnit(source []byte, target string, strip bool, windowsGUI bool) driver.BackendResult {
	return backend.compile(source, driver.BackendCompileOptions{
		Target: target, Strip: strip, WindowsGUI: windowsGUI,
	})
}

func (backend *SeededBackend) CompileUnitWithArena(source []byte, target string, strip bool, windowsGUI bool, arenaSize int) driver.BackendResult {
	return backend.compile(source, driver.BackendCompileOptions{
		Target: target, Strip: strip, WindowsGUI: windowsGUI, ArenaSize: arenaSize,
	})
}

func (backend *SeededBackend) CompileUnitWithOptions(source []byte, options driver.BackendCompileOptions) driver.BackendResult {
	return backend.compile(source, options)
}

func (backend *SeededBackend) compile(source []byte, options driver.BackendCompileOptions) driver.BackendResult {
	compileTarget := backend.compileTarget(options)
	prepared := backend.prepare(compileTarget)
	if !prepared.Ok {
		return driver.BackendResult{Diagnostic: prepared.Diagnostic}
	}
	binding, ok := unit.ReadTargetBinding(source)
	if !ok || binding.Target != prepared.Artifact.Descriptor.Name ||
		binding.Definition != string(prepared.Artifact.Descriptor.Definition[:]) ||
		binding.DescriptorVersion != prepared.Artifact.Descriptor.Version {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-007",
			Message: "unit target binding does not match the derived backend",
		}}
	}
	request := Request{Protocol: ProtocolVersion, Unit: source, Options: options}
	if prepared.Artifact.Host == backendvm32.Target {
		return runVMCompiler(prepared.Artifact, request)
	}
	var result driver.BackendResult
	if prepared.ExecutablePath != "" {
		result = runNativeCompiler(prepared.ExecutablePath, prepared.Artifact, request)
	} else {
		result = runNativeCompilerPayload(prepared.Artifact, request)
	}
	if result.Ok || result.Diagnostic.Code != "RENVO-BACKEND-011" {
		return result
	}
	fallback := backend.prepareVMFallback(compileTarget)
	if !fallback.Ok {
		return result
	}
	return runVMCompiler(fallback.Artifact, request)
}

func (backend *SeededBackend) compileTarget(options driver.BackendCompileOptions) string {
	if backend.builtin && options.Mode == driver.ModeKernelModule {
		return "linux-kernel/amd64"
	}
	return options.Target
}

func (backend *SeededBackend) prepareVMFallback(target string) Prepared {
	backend.prepareMu.Lock()
	defer backend.prepareMu.Unlock()
	if backend.vmPrepared.Ok && backend.vmFor == target {
		return backend.vmPrepared
	}
	var resolved rtg.ResolveResult
	if backend.builtin {
		resolved = backendbuiltin.Resolve(target)
	} else {
		source, err := os.ReadFile(backend.path)
		if err != nil || rtgb.IsArtifact(source) {
			return Prepared{}
		}
		resolved = rtg.ResolveDefinitions(rtg.ParseImports(source, backend.path, filesystemImportLoader{}))
	}
	if !resolved.Ok {
		return Prepared{}
	}
	generated := rtg.GeneratePreparedBackend(resolved, target)
	if !generated.Ok {
		return Prepared{}
	}
	unitSource, diagnostic, ok := compilerProgramUnit(
		generated, backendvm32.Target, seededVMCompilerArenaSize, backend.stdRoot)
	if !ok {
		return Prepared{Diagnostic: diagnostic}
	}
	compiled := (backendvm32.Backend{}).CompileUnitWithArena(
		unitSource, backendvm32.Target, true, false, seededVMCompilerArenaSize)
	if !compiled.Ok {
		return Prepared{Diagnostic: compiled.Diagnostic}
	}
	backend.vmPrepared = Prepared{
		Artifact: seededArtifact(generated.Descriptor, backendvm32.Target, compiled.Binary), Ok: true,
	}
	backend.vmFor = target
	return backend.vmPrepared
}

func (backend *SeededBackend) prepare(target string) Prepared {
	backend.prepareMu.Lock()
	defer backend.prepareMu.Unlock()
	if backend.prepared.Ok && backend.preparedFor == target {
		return backend.prepared
	}
	if !backend.builtin {
		source, err := os.ReadFile(backend.path)
		if err != nil {
			return prepareFailure("RENVO-RTG-011", "could not read backend "+backend.path)
		}
		if rtgb.IsArtifact(source) {
			backend.prepared = loadSeeded(source, target)
			backend.preparedFor = target
			return backend.prepared
		}
		resolved := rtg.ResolveDefinitions(rtg.ParseImports(source, backend.path, filesystemImportLoader{}))
		backend.prepared = prepareSeeded(resolved, target, backend.stdRoot, backend.cacheDir)
		backend.preparedFor = target
		return backend.prepared
	}
	backend.prepared = prepareSeeded(backendbuiltin.Resolve(target), target, backend.stdRoot, backend.cacheDir)
	backend.preparedFor = target
	return backend.prepared
}

func prepareSeeded(resolved rtg.ResolveResult, target string, stdRoot string, cacheDir string) Prepared {
	if !resolved.Ok {
		message := "backend definition did not resolve"
		if len(resolved.Diagnostics) != 0 {
			message = resolved.Diagnostics[0].Message
		}
		return prepareFailure("RENVO-RTG-001", message)
	}
	generated := rtg.GeneratePreparedBackend(resolved, target)
	if !generated.Ok {
		return prepareFailure("RENVO-RTG-002", generated.Diagnostics[0].Message)
	}
	host := hostTarget()
	var hostGenerated rtg.GenerateResult
	var hostDefinition [32]byte
	if host != "" {
		hostResolved := backendbuiltin.Resolve(host)
		if !hostResolved.Ok {
			return prepareFailure("RENVO-RTG-003", "host backend definition is unavailable")
		}
		hostGenerated = rtg.GeneratePreparedBackend(hostResolved, host)
		if !hostGenerated.Ok {
			return prepareFailure("RENVO-RTG-003", hostGenerated.Diagnostics[0].Message)
		}
		hostDefinition = hostGenerated.Descriptor.Definition
	}
	key := seededCacheKey("compiler", seededKernelVersion, generated.Descriptor, host, hostDefinition)
	if cacheDir != "" {
		if prepared, found := loadNativeCompilerCache(cacheDir, key, generated.Descriptor, host); found {
			return prepared
		}
	}
	artifact, diagnostic, ok := deriveCompiler(generated, hostGenerated, host, stdRoot, cacheDir)
	if !ok && host != "" {
		artifact, diagnostic, ok = deriveCompiler(generated, rtg.GenerateResult{}, "", stdRoot, "")
	}
	if !ok {
		return Prepared{Diagnostic: diagnostic}
	}
	if cacheDir != "" {
		if prepared, err := storeNativeCompilerCache(cacheDir, key, artifact); err == nil {
			return prepared
		}
	}
	encoded, encodedOK := rtgb.Encode(artifact)
	if !encodedOK {
		return prepareFailure("RENVO-RTG-006", "derived backend artifact is invalid")
	}
	return Prepared{Artifact: artifact, Encoded: encoded, Ok: true}
}

func deriveCompiler(generated rtg.GenerateResult, hostGenerated rtg.GenerateResult, host string, stdRoot string, cacheDir string) (rtgb.Artifact, driver.Diagnostic, bool) {
	if host == "" {
		unitSource, diagnostic, ok := compilerProgramUnit(generated, backendvm32.Target, seededVMCompilerArenaSize, stdRoot)
		if !ok {
			return rtgb.Artifact{}, diagnostic, false
		}
		compiled := (backendvm32.Backend{}).CompileUnitWithArena(unitSource, backendvm32.Target, true, false, seededVMCompilerArenaSize)
		if !compiled.Ok {
			return rtgb.Artifact{}, compiled.Diagnostic, false
		}
		return seededArtifact(generated.Descriptor, backendvm32.Target, compiled.Binary), driver.Diagnostic{}, true
	}
	emitter, diagnostic, ok := cachedHostEmitter(hostGenerated, stdRoot, cacheDir)
	if !ok {
		return rtgb.Artifact{}, diagnostic, false
	}
	hostUnit, diagnostic, ok := compilerProgramUnit(generated, host, seededCompilerArenaSize, stdRoot)
	if !ok {
		return rtgb.Artifact{}, diagnostic, false
	}
	emitterArgs := compilerRequestArgs(host, driver.BackendCompileOptions{Target: host, Strip: true}, "-", "-")
	emitterArgs = append([]string{"renvo-host-emitter"}, emitterArgs...)
	result := vm.RunConfig(emitter, vm.Config{
		Limits: vm.Limits{Steps: seededVMStepLimit, Memory: seededVMMemoryLimit},
		Args:   emitterArgs,
		Env:    []string{"PATH=/vm", "PWD=/"}, Stdin: hostUnit,
	})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || len(result.Output) == 0 {
		message := "VM host emitter failed"
		if len(result.Stderr) != 0 {
			message += ": " + string(result.Stderr)
		}
		return rtgb.Artifact{}, driver.Diagnostic{Phase: "backend-prepare", Code: "RENVO-RTG-005", Message: message}, false
	}
	if !validHostExecutable(result.Output, host) {
		return rtgb.Artifact{}, driver.Diagnostic{Phase: "backend-prepare", Code: "RENVO-RTG-005", Message: "VM host emitter produced an invalid host executable"}, false
	}
	return seededArtifact(generated.Descriptor, host, result.Output), driver.Diagnostic{}, true
}

func cachedHostEmitter(generated rtg.GenerateResult, stdRoot string, cacheDir string) ([]byte, driver.Diagnostic, bool) {
	key := seededCacheKey("host-emitter", seededHostEmitterKernel, generated.Descriptor, backendvm32.Target, [32]byte{})
	cache := FileCache{Directory: cacheDir}
	if cacheDir != "" {
		if trustedCachedFile(filepath.Join(cacheDir, key+".rtgb"), false) {
			encoded, found := cache.Load(key)
			artifact, ok := rtgb.Decode(encoded)
			if found && ok && artifact.Kernel == seededHostEmitterKernel && artifact.Host == backendvm32.Target &&
				artifact.Descriptor.Name == generated.Descriptor.Name &&
				artifact.Descriptor.Definition == generated.Descriptor.Definition &&
				artifact.Generator == rtg.GeneratorVersion && artifact.Unit == unit.Version &&
				artifact.Protocol == ProtocolVersion && artifact.Optimization == OptimizationVersion {
				return artifact.Payload, driver.Diagnostic{}, true
			}
		}
	}
	vmUnit, diagnostic, ok := compilerProgramUnit(generated, backendvm32.Target, seededVMCompilerArenaSize, stdRoot)
	if !ok {
		return nil, diagnostic, false
	}
	compiled := (backendvm32.Backend{}).CompileUnitWithArena(vmUnit, backendvm32.Target, true, false, seededVMCompilerArenaSize)
	if !compiled.Ok {
		return nil, compiled.Diagnostic, false
	}
	artifact := seededArtifact(generated.Descriptor, backendvm32.Target, compiled.Binary)
	artifact.Kernel = seededHostEmitterKernel
	encoded, encodedOK := rtgb.Encode(artifact)
	if !encodedOK {
		return nil, driver.Diagnostic{Phase: "backend-prepare", Code: "RENVO-RTG-006", Message: "host emitter artifact is invalid"}, false
	}
	if cacheDir != "" {
		_ = cache.Store(key, encoded)
	}
	return compiled.Binary, driver.Diagnostic{}, true
}

func compilerProgramUnit(generated rtg.GenerateResult, target string, arenaSize int, stdRoot string) ([]byte, driver.Diagnostic, bool) {
	sources, names, err := preparationSources("", generated)
	if err != nil {
		return nil, driver.Diagnostic{Phase: "backend-prepare", Code: "RENVO-RTG-004", Message: err.Error()}, false
	}
	args := []string{"-t", target, "-arena-size", decimal(arenaSize), "-emit-unit", "-o", "-"}
	args = append(args, names...)
	built := driver.BuildUnit(args, "/backend", stdRoot, sources)
	if !built.Ok {
		return nil, built.Diagnostic, false
	}
	return built.Unit, driver.Diagnostic{}, true
}

func seededArtifact(descriptor rtg.TargetDescriptor, host string, payload []byte) rtgb.Artifact {
	return rtgb.Artifact{
		Descriptor: descriptor, Host: host, Generator: rtg.GeneratorVersion,
		Kernel: seededKernelVersion, Protocol: ProtocolVersion, Unit: unit.Version,
		Optimization: OptimizationVersion, Payload: payload,
	}
}

func seededCompatible(artifact rtgb.Artifact, descriptor rtg.TargetDescriptor, host string) bool {
	expectedHost := host
	if expectedHost == "" {
		expectedHost = backendvm32.Target
	}
	return artifact.Descriptor.Name == descriptor.Name && artifact.Descriptor.Definition == descriptor.Definition &&
		artifact.Descriptor.Version == descriptor.Version && artifact.Host == expectedHost &&
		artifact.Generator == rtg.GeneratorVersion && artifact.Kernel == seededKernelVersion &&
		artifact.Protocol == ProtocolVersion && artifact.Unit == unit.Version &&
		artifact.Optimization == OptimizationVersion
}

func loadSeeded(source []byte, target string) Prepared {
	artifact, ok := rtgb.Decode(source)
	if !ok || artifact.Kernel != seededKernelVersion || artifact.Generator != rtg.GeneratorVersion ||
		artifact.Descriptor.Version != rtg.DescriptorVersion ||
		artifact.Protocol != ProtocolVersion || artifact.Unit != unit.Version ||
		artifact.Optimization != OptimizationVersion {
		return prepareFailure("RENVO-RTG-009", "derived backend is incompatible with this compiler")
	}
	expectedHost := hostTarget()
	if expectedHost == "" {
		expectedHost = backendvm32.Target
	}
	if artifact.Host != expectedHost ||
		artifact.Descriptor.Name != target && !contains(artifact.Descriptor.Aliases, target) {
		return prepareFailure("RENVO-RTG-010", "derived backend target or host does not match this request")
	}
	return Prepared{Artifact: artifact, Encoded: source, Ok: true}
}

func seededCacheKey(kind string, kernel int, descriptor rtg.TargetDescriptor, host string, hostDefinition [32]byte) string {
	identity := kind + "-" + rtg.HashText(descriptor.Definition) + "-" + encodedName(descriptor.Name) +
		"-" + encodedName(host) + "-h" + rtg.HashText(hostDefinition) +
		"-d" + decimal(descriptor.Version) + "-g" + decimal(rtg.GeneratorVersion) + "-k" + decimal(kernel) +
		"-u" + decimal(unit.Version) + "-p" + decimal(ProtocolVersion) +
		"-o" + decimal(OptimizationVersion) + "-s" + decimal(backendvm32.SeedVersion) +
		"-a" + decimal(seededCompilerArenaSize) + "-v" + decimal(seededVMCompilerArenaSize) +
		"-c" + backendvm32.CompilerSourceDigest
	digest := sha256.Sum256([]byte(identity))
	return kind + "-" + rtg.HashText(digest)
}

func runVMCompiler(artifact rtgb.Artifact, request Request) driver.BackendResult {
	if !compatibleRequest(artifact, request) {
		return incompatibleProtocol()
	}
	compilerArgs := compilerRequestArgs(artifact.Descriptor.Name, request.Options, "-", "-")
	compilerArgs = append([]string{"renvo-vm-compiler"}, compilerArgs...)
	result := vm.RunConfig(artifact.Payload, vm.Config{
		Limits: vm.Limits{Steps: seededVMStepLimit, Memory: seededVMMemoryLimit},
		Args:   compilerArgs,
		Env:    []string{"PATH=/vm", "PWD=/"}, Stdin: request.Unit,
	})
	if result.Trap != vm.TrapNone || result.ExitCode != 0 || len(result.Output) == 0 {
		message := "VM compiler execution failed"
		if len(result.Stderr) != 0 {
			message += ": " + string(result.Stderr)
		}
		return driver.BackendResult{Diagnostic: driver.Diagnostic{Phase: "backend", Code: "RENVO-BACKEND-009", Message: message}}
	}
	return driver.BackendResult{Binary: result.Output, Ok: true}
}
