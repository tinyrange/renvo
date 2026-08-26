//go:build !renvo

package backendjit

import (
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"renvo.dev/internal/driver"
	"renvo.dev/internal/linkedimage"
	"renvo.dev/internal/load"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgb"
	"renvo.dev/internal/runimage"
	"renvo.dev/internal/unit"
	"renvo.dev/std/vm"
)

type Backend struct {
	path          string
	backendRoot   string
	stdRoot       string
	cacheDir      string
	bootstrap     driver.Backend
	runner        Runner
	prepared      Prepared
	assemblyCache map[[32]byte][]byte
}

func New(path string, backendRoot string, stdRoot string, cacheDir string, bootstrap driver.Backend) *Backend {
	return NewWithRunner(path, backendRoot, stdRoot, cacheDir, bootstrap, ProcessRunner{})
}

// NewWithRunner separates backend preparation from execution for embedders
// that cannot or do not want to spawn the prepared native image themselves.
func NewWithRunner(path string, backendRoot string, stdRoot string, cacheDir string, bootstrap driver.Backend, runner Runner) *Backend {
	return &Backend{
		path: path, backendRoot: backendRoot, stdRoot: stdRoot,
		cacheDir: cacheDir, bootstrap: bootstrap, runner: runner,
	}
}

type Request struct {
	Protocol int
	Unit     []byte
	Source   []byte
	Options  driver.BackendCompileOptions
}

type Runner interface {
	Run(artifact rtgb.Artifact, request Request) driver.BackendResult
}

type ProcessRunner struct{}

// Native prepared compiler images execute in this process and therefore share
// its low-level runtime state. Keep their complete request lifetimes isolated;
// concurrent test and IDE requests otherwise race while opening protocol files.
var processRunnerMu sync.Mutex

func (b *Backend) SupportsRTGAssembly() bool { return true }

func (b *Backend) CompileUnit(source []byte, target string, strip bool, windowsGUI bool) driver.BackendResult {
	return b.CompileUnitWithArena(source, target, strip, windowsGUI, 0)
}

func (b *Backend) CompileUnitWithArena(source []byte, target string, strip bool, windowsGUI bool, arenaSize int) driver.BackendResult {
	return b.compile(source, driver.BackendCompileOptions{
		Target: target, Strip: strip, WindowsGUI: windowsGUI, ArenaSize: arenaSize,
	})
}

func (b *Backend) CompileUnitWithOptions(source []byte, options driver.BackendCompileOptions) driver.BackendResult {
	return b.compile(source, options)
}

// CompileSourceWithArena runs a standalone backend-corpus program directly
// through the prepared compiler. Unlike CompileUnit, it intentionally bypasses
// frontend checking, linking, and pruning.
func (b *Backend) CompileSourceWithArena(source []byte, target string, strip bool, arenaSize int) driver.BackendResult {
	prepared := b.prepare(target)
	if !prepared.Ok {
		return driver.BackendResult{Diagnostic: prepared.Diagnostic}
	}
	if b.runner == nil {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-009", Message: "prepared backend runner is unavailable",
		}}
	}
	return b.runner.Run(prepared.Artifact, Request{
		Protocol: ProtocolVersion,
		Source:   source,
		Options: driver.BackendCompileOptions{
			Target: target, Strip: strip, ArenaSize: arenaSize,
		},
	})
}

func (b *Backend) compile(source []byte, options driver.BackendCompileOptions) driver.BackendResult {
	prepared := b.prepare(options.Target)
	if !prepared.Ok {
		return driver.BackendResult{Diagnostic: prepared.Diagnostic}
	}
	binding, ok := unit.ReadTargetBinding(source)
	if !ok || binding.Target != prepared.Artifact.Descriptor.Name ||
		binding.Definition != string(prepared.Artifact.Descriptor.Definition[:]) ||
		binding.DescriptorVersion != prepared.Artifact.Descriptor.Version {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-007",
			Message: "unit target binding does not match the prepared backend",
		}}
	}
	if _, assemblyBindings, hasAssembly := readRTGAssembly(source); hasAssembly && len(assemblyBindings) != 0 {
		var evaluated driver.BackendResult
		source, evaluated = b.evaluateRTGAssembly(source, prepared)
		if !evaluated.Ok {
			return evaluated
		}
	}
	if b.runner == nil {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-009", Message: "prepared backend runner is unavailable",
		}}
	}
	return b.runner.Run(prepared.Artifact, Request{
		Protocol: ProtocolVersion,
		Unit:     source,
		Options:  options,
	})
}

func (b *Backend) evaluateRTGAssembly(source []byte, prepared Prepared) ([]byte, driver.BackendResult) {
	sources, bindings, ok := readRTGAssembly(source)
	if !ok || !prepared.Resolved.Ok {
		return nil, driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "rtgasm", Code: "RENVO-RTGASM-002", Message: "prepared backend cannot evaluate RTGASM source",
		}}
	}
	code := make([][]byte, len(bindings))
	documents := make([]rtg.AssemblyDocument, len(sources))
	for i := 0; i < len(sources); i++ {
		documents[i] = rtg.ParseAssembly(sources[i].Source, sources[i].Path)
		if !documents[i].Ok {
			return nil, driver.BackendResult{Diagnostic: driver.Diagnostic{
				Phase: "rtgasm", Code: "RENVO-RTGASM-003", Message: documents[i].Diagnostics[0].Message,
				Path: sources[i].Path, Start: documents[i].Diagnostics[0].Span.Start.Offset,
			}}
		}
	}
	for i := 0; i < len(bindings); i++ {
		if len(bindings[i].Code) != 0 {
			code[i] = bindings[i].Code
			continue
		}
		binding := bindings[i]
		if binding.Source < 0 || binding.Source >= len(documents) || binding.Entry < 0 || binding.Entry >= len(documents[binding.Source].Entries) {
			return nil, rtgAssemblyBackendFailure("RENVO-RTGASM-004", "RTGASM binding is invalid")
		}
		cacheInput := make([]byte, 0, len(prepared.Artifact.Descriptor.Definition)+len(sources[binding.Source].Source)+8)
		cacheInput = append(cacheInput, prepared.Artifact.Descriptor.Definition[:]...)
		cacheInput = append(cacheInput, sources[binding.Source].Source...)
		cacheInput = append(cacheInput, byte(binding.Entry), byte(binding.Entry>>8), byte(binding.Entry>>16), byte(binding.Entry>>24))
		cacheKey := sha256.Sum256(cacheInput)
		if cached := b.assemblyCache[cacheKey]; len(cached) != 0 {
			code[i] = append([]byte(nil), cached...)
			continue
		}
		generated := rtg.GenerateAssemblyEvaluator(prepared.Resolved, prepared.Artifact.Descriptor.Name, documents[binding.Source], binding.Entry)
		if !generated.Ok {
			return nil, rtgAssemblyBackendFailure("RENVO-RTGASM-005", generated.Diagnostics[0].Message)
		}
		files, names, err := assemblyEvaluationSources(generated)
		if err != nil {
			return nil, rtgAssemblyBackendFailure("RENVO-RTGASM-006", err.Error())
		}
		args := []string{"-s", "-emit-image", "-t", "vm/vm32", "-arena-size", "67108864", "-o", "-"}
		args = append(args, names...)
		compiled := driver.CompileUnit(args, "/backend", b.stdRoot, files, b.bootstrap)
		if !compiled.Ok {
			entry := documents[binding.Source].Entries[binding.Entry]
			return nil, driver.BackendResult{Diagnostic: driver.Diagnostic{
				Phase: "rtgasm", Code: "RENVO-RTGASM-011",
				Message: "RTGASM fragment does not compile against the selected target: " + compiled.Diagnostic.Message,
				Path:    sources[binding.Source].Path, Start: entry.Span.Start.Offset,
			}}
		}
		image, err := linkedimage.Decode(compiled.Binary)
		if err != nil || image.Target != "vm/vm32" {
			return nil, rtgAssemblyBackendFailure("RENVO-RTGASM-007", "RTGASM evaluator image is invalid")
		}
		run := vm.RunConfig(image.Native, vm.Config{Limits: vm.Limits{Steps: 500000000, Memory: 96 * 1024 * 1024}})
		if run.Trap != vm.TrapNone || run.ExitCode != 0 || len(run.Output) == 0 {
			return nil, rtgAssemblyBackendFailure("RENVO-RTGASM-008", "RTGASM evaluator failed")
		}
		code[i] = run.Output
		if b.assemblyCache == nil {
			b.assemblyCache = make(map[[32]byte][]byte)
		}
		b.assemblyCache[cacheKey] = append([]byte(nil), run.Output...)
	}
	evaluated, ok := attachRTGAssemblyCode(source, code)
	if !ok {
		return nil, rtgAssemblyBackendFailure("RENVO-RTGASM-009", "could not attach evaluated RTGASM code")
	}
	return evaluated, driver.BackendResult{Ok: true}
}

// EvaluateRTGAssembly evaluates preserved project assembly against an already
// resolved target. Sandboxed frontends use this before invoking a separately
// prepared compiler image.
func EvaluateRTGAssembly(source []byte, resolved rtg.ResolveResult, descriptor rtg.TargetDescriptor, stdRoot string, bootstrap driver.Backend) ([]byte, driver.BackendResult) {
	_, bindings, hasAssembly := readRTGAssembly(source)
	if !hasAssembly || len(bindings) == 0 {
		return source, driver.BackendResult{Ok: true}
	}
	b := Backend{stdRoot: stdRoot, bootstrap: bootstrap}
	prepared := Prepared{Resolved: resolved, Artifact: rtgb.Artifact{Descriptor: descriptor}}
	return b.evaluateRTGAssembly(source, prepared)
}

func assemblyEvaluationSources(generated rtg.GenerateResult) ([]load.SourceFile, []string, error) {
	sources, names, err := preparationSources("", generated)
	if err != nil {
		return nil, nil, err
	}
	for i := 0; i < len(sources); i++ {
		if filepath.Base(sources[i].Path) != "compiler_main.go" {
			continue
		}
		sources[i].Src = bytes.Replace(sources[i].Src, []byte("func appMain("), []byte("func renvoCompilerMain("), 1)
		break
	}
	return sources, names, nil
}

func rtgAssemblyBackendFailure(code string, message string) driver.BackendResult {
	return driver.BackendResult{Diagnostic: driver.Diagnostic{Phase: "rtgasm", Code: code, Message: message}}
}

func (b *Backend) prepare(target string) Prepared {
	if b.prepared.Ok {
		if b.prepared.Artifact.Descriptor.Name == target || contains(b.prepared.Artifact.Descriptor.Aliases, target) {
			return b.prepared
		}
		return prepareFailure("RENVO-RTG-010", "prepared backend target does not match "+target)
	}
	source, err := os.ReadFile(b.path)
	if err != nil {
		return prepareFailure("RENVO-RTG-011", "could not read backend "+b.path)
	}
	if rtgb.IsArtifact(source) {
		b.prepared = Load(source)
	} else {
		workDir, _ := os.Getwd()
		b.prepared = Prepare(PrepareConfig{
			Definition: source, Filename: b.path, Target: target,
			ImportLoader: filesystemImportLoader{},
			BackendRoot:  b.backendRoot, WorkDir: workDir, StdRoot: b.stdRoot,
			CacheDir: b.cacheDir, Bootstrap: b.bootstrap,
		})
	}
	return b.prepared
}

type filesystemImportLoader struct{}

func (filesystemImportLoader) LoadImport(
	importingFilename string, importPath string,
) rtg.ImportSource {
	path := filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath))
	source, err := os.ReadFile(path)
	return rtg.ImportSource{Source: source, Filename: path, Ok: err == nil}
}

func (ProcessRunner) Run(artifact rtgb.Artifact, request Request) driver.BackendResult {
	processRunnerMu.Lock()
	defer processRunnerMu.Unlock()
	if request.Protocol != ProtocolVersion || artifact.Protocol != ProtocolVersion ||
		artifact.Unit != unit.Version || artifact.Optimization != OptimizationVersion {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-008", Message: "prepared backend protocol is incompatible",
		}}
	}
	image, err := linkedimage.Decode(artifact.Payload)
	if err != nil || image.Target != artifact.Host {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-008", Message: "prepared backend payload is invalid",
		}}
	}
	tempDir, err := os.MkdirTemp("", "renvo-backend-jit-*")
	if err != nil {
		return backendIOError("create protocol directory")
	}
	defer func() { _ = os.RemoveAll(tempDir) }()
	inputName := "input.rtgu"
	input := request.Unit
	if len(request.Source) != 0 {
		inputName = "input.go"
		input = request.Source
	}
	inputPath := filepath.Join(tempDir, inputName)
	outputPath := filepath.Join(tempDir, "output.bin")
	if err = os.WriteFile(inputPath, input, 0o600); err != nil {
		return backendIOError("write protocol input")
	}
	args := []string{"-t", artifact.Descriptor.Name}
	options := request.Options
	if options.Strip {
		args = append(args, "-s")
	}
	if options.WindowsGUI {
		args = append(args, "-windows-gui")
	}
	if options.EmitImage {
		args = append(args, "-emit-image")
	}
	if options.ObjectFile || options.Mode == driver.ModeObject {
		args = append(args, "-object")
	}
	if options.ArenaSize > 0 {
		args = append(args, "-arena-size", decimal(options.ArenaSize))
	}
	if options.Output != "" {
		args = append(args, "-module-name", options.Output)
	}
	if options.ModuleLicense != "" {
		args = append(args, "-module-license", options.ModuleLicense)
	}
	args = append(args, "-o", outputPath, inputPath)
	executed := runimage.Run(image, "renvo-prepared-backend", args, os.Environ(), os.Stdin,
		os.Stdout, os.Stderr)
	if executed.Err != nil || executed.ExitCode != 0 {
		message := "prepared backend execution failed with exit code " + strconv.Itoa(executed.ExitCode)
		if executed.Err != nil {
			message += ": " + executed.Err.Error()
		}
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-009", Message: message,
		}}
	}
	output, err := os.ReadFile(outputPath)
	if err != nil || len(output) == 0 {
		return backendIOError("read protocol output")
	}
	return driver.BackendResult{Binary: output, Ok: true}
}

func backendIOError(action string) driver.BackendResult {
	return driver.BackendResult{Diagnostic: driver.Diagnostic{
		Phase: "backend", Code: "RENVO-BACKEND-010", Message: action + " failed",
	}}
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}
