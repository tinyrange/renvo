//go:build !renvo

// Package backendjit prepares external RTG definitions with Renvo's built-in
// host backend and executes the resulting backend image.
package backendjit

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
	"renvo.dev/internal/rbe"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgb"
	"renvo.dev/internal/unit"
)

const (
	KernelVersion            = 1
	ProtocolVersion          = 2
	OptimizationVersion      = 1
	preparedBackendArenaSize = 1073741824
)

type PrepareConfig struct {
	Definition   []byte
	Filename     string
	ImportLoader rtg.ImportLoader
	Target       string
	BackendRoot  string
	WorkDir      string
	StdRoot      string
	CacheDir     string
	Cache        ArtifactCache
	Bootstrap    driver.Backend
	// HostTarget overrides the executable format used for the prepared compiler.
	// Empty selects the current native host. Sandboxed embedders use vm/vm32.
	HostTarget string
	// ArenaSize controls the prepared compiler's own arena. Zero uses the native
	// process default; VM embedders should choose a bounded value they can host.
	ArenaSize int
}

// ArtifactCache lets embedders keep prepared artifacts outside the host
// filesystem. Keys are content and compiler compatibility identities.
type ArtifactCache interface {
	Load(key string) ([]byte, bool)
	Store(key string, source []byte) error
}

type Prepared struct {
	Artifact   rtgb.Artifact
	Resolved   rtg.ResolveResult
	Encoded    []byte
	CachePath  string
	CacheHit   bool
	Diagnostic driver.Diagnostic
	Ok         bool
}

func Prepare(config PrepareConfig) Prepared {
	bundle := rbe.Parse(config.Definition)
	if !bundle.Ok {
		return Prepared{Diagnostic: driver.Diagnostic{
			Phase: "rbe", Code: "RENVO-RBE-001", Message: bundle.Message,
			Path: config.Filename, Start: bundle.Offset,
		}}
	}
	enablement := sha256.Sum256(config.Definition)
	resolved := rtg.ResolveDefinitions(rtg.ParseImports(
		bundle.Definition, config.Filename, config.ImportLoader))
	if !resolved.Ok {
		return prepareFailure("RENVO-RTG-001", resolved.Diagnostics[0].Message)
	}
	host := config.HostTarget
	if host == "" {
		host = hostTarget()
	}
	if host == "" {
		return prepareFailure("RENVO-RTG-003", "this host cannot prepare native backends")
	}
	arenaSize := config.ArenaSize
	if arenaSize == 0 {
		arenaSize = preparedBackendArenaSize
	}
	key := ""
	cachePath := ""
	cache := config.Cache
	if cache == nil && config.CacheDir != "" {
		cache = FileCache{Directory: config.CacheDir}
	}
	descriptor, hasDescriptor := resolvedTargetDescriptor(resolved, config.Target)
	if hasDescriptor {
		key = cacheKeyForEnablement(descriptor, host, arenaSize, enablement)
		if config.CacheDir != "" {
			cachePath = filepath.Join(config.CacheDir, key+".rtgb")
		}
	}
	if cache != nil && key != "" {
		if source, found := cache.Load(key); found {
			if artifact, ok := rtgb.Decode(source); ok && compatible(artifact, descriptor, host, enablement) {
				return Prepared{Artifact: artifact, Resolved: resolved, Encoded: source, CachePath: cachePath, CacheHit: true, Ok: true}
			}
		}
	}
	generated := rtg.GeneratePreparedBackend(resolved, config.Target)
	if !generated.Ok {
		return prepareFailure("RENVO-RTG-002", generated.Diagnostics[0].Message)
	}
	if key == "" {
		key = cacheKeyForEnablement(generated.Descriptor, host, arenaSize, enablement)
		if config.CacheDir != "" {
			cachePath = filepath.Join(config.CacheDir, key+".rtgb")
		}
	}
	sources, names, err := preparationSources(config.BackendRoot, generated)
	if err != nil {
		return prepareFailure("RENVO-RTG-004", err.Error())
	}
	args := make([]string, 0, len(names)+9)
	args = append(args, "-s", "-emit-image", "-t", host, "-arena-size", decimal(arenaSize), "-o", "-")
	args = append(args, names...)
	compiled := driver.CompileUnit(args, "/backend", config.StdRoot, sources, config.Bootstrap)
	if !compiled.Ok {
		diagnostic := compiled.Diagnostic
		if !diagnostic.Valid() {
			diagnostic = driver.Diagnostic{Phase: "backend-prepare", Code: "RENVO-RTG-005", Message: "generated backend did not compile"}
		}
		return Prepared{Diagnostic: diagnostic}
	}
	artifact := rtgb.Artifact{
		Descriptor:      generated.Descriptor,
		Host:            host,
		Generator:       rtg.GeneratorVersion,
		Kernel:          KernelVersion,
		Protocol:        ProtocolVersion,
		Unit:            unit.Version,
		Optimization:    OptimizationVersion,
		Enablement:      enablement,
		DefinitionFiles: rtg.SourceBundle(resolved.Document),
		LibraryFiles:    bundle.Files,
		Payload:         compiled.Binary,
	}
	encoded, ok := rtgb.Encode(artifact)
	if !ok {
		return prepareFailure("RENVO-RTG-006", "prepared backend artifact is invalid")
	}
	if cache != nil {
		if err := cache.Store(key, encoded); err != nil {
			return prepareFailure("RENVO-RTG-007", err.Error())
		}
	}
	return Prepared{Artifact: artifact, Resolved: resolved, Encoded: encoded, CachePath: cachePath, Ok: true}
}

func resolvedTargetDescriptor(resolved rtg.ResolveResult, name string) (rtg.TargetDescriptor, bool) {
	for i := range resolved.Targets {
		descriptor := resolved.Targets[i].Descriptor
		if descriptor.Name == name || contains(descriptor.Aliases, name) {
			return descriptor, true
		}
	}
	return rtg.TargetDescriptor{}, false
}

// FileCache is the process-host adapter for content-addressed artifacts.
type FileCache struct {
	Directory string
}

func (cache FileCache) Load(key string) ([]byte, bool) {
	if cache.Directory == "" {
		return nil, false
	}
	source, err := os.ReadFile(filepath.Join(cache.Directory, key+".rtgb"))
	return source, err == nil
}

func (cache FileCache) Store(key string, source []byte) error {
	if cache.Directory == "" {
		return fmt.Errorf("backend cache directory is empty")
	}
	return publish(filepath.Join(cache.Directory, key+".rtgb"), source)
}

func Load(source []byte) Prepared {
	artifact, ok := rtgb.Decode(source)
	if !ok {
		return prepareFailure("RENVO-RTG-008", "invalid prepared backend artifact")
	}
	host := hostTarget()
	if artifact.Host != host || artifact.Generator != rtg.GeneratorVersion ||
		artifact.Descriptor.Version != rtg.DescriptorVersion ||
		artifact.Kernel != KernelVersion || artifact.Protocol != ProtocolVersion ||
		artifact.Unit != unit.Version || artifact.Optimization != OptimizationVersion {
		return prepareFailure("RENVO-RTG-009", "prepared backend is incompatible with this compiler or host")
	}
	var resolved rtg.ResolveResult
	if len(artifact.DefinitionFiles) != 0 {
		root := artifact.DefinitionFiles[0]
		resolved = rtg.ResolveDefinitions(rtg.ParseImports(root.Source, root.Filename,
			artifactDefinitionLoader{files: artifact.DefinitionFiles}))
		if !resolved.Ok {
			message := "prepared backend carries an invalid closed definition"
			if len(resolved.Diagnostics) != 0 {
				message += ": " + resolved.Diagnostics[0].Message
			}
			return prepareFailure("RENVO-RTG-009", message)
		}
	}
	return Prepared{Artifact: artifact, Resolved: resolved, Encoded: source, Ok: true}
}

type artifactDefinitionLoader struct{ files []rtg.ImportSource }

func (loader artifactDefinitionLoader) LoadImport(importingFilename string, importPath string) rtg.ImportSource {
	wanted := load.JoinPath(load.DirPath(importingFilename), importPath)
	for i := 0; i < len(loader.files); i++ {
		if loader.files[i].Filename == wanted {
			return loader.files[i]
		}
	}
	return rtg.ImportSource{}
}

func preparationSources(backendRoot string, generated rtg.GenerateResult) ([]load.SourceFile, []string, error) {
	_ = backendRoot
	excluded := excludedFamily(generated.Descriptor)
	sources := []load.SourceFile{{Path: "/backend/go.mod", Src: []byte("module renvo.dev/prepared-backend\n")}}
	var names []string
	for i := 0; i < backendcompiled.CompilerSourceCount; i++ {
		name, source, ok := backendcompiled.CompilerSource(i)
		if !ok {
			return nil, nil, fmt.Errorf("decompress backend kernel source %d", i)
		}
		if name == "" || excluded[name] {
			continue
		}
		path := load.JoinPath("/backend", name)
		sources = append(sources, load.SourceFile{Path: path, Src: []byte(source)})
		names = append(names, path)
	}
	generatedPath := "/backend/compiler_rtg_prepared_impl.go"
	sources = append(sources, load.SourceFile{Path: generatedPath, Src: generated.Source})
	names = append(names, generatedPath)
	return sources, names, nil
}

func compatible(artifact rtgb.Artifact, descriptor rtg.TargetDescriptor, host string, enablement [32]byte) bool {
	return artifact.Descriptor.Name == descriptor.Name &&
		artifact.Descriptor.Definition == descriptor.Definition &&
		artifact.Descriptor.Version == descriptor.Version &&
		artifact.Host == host &&
		artifact.Generator == rtg.GeneratorVersion &&
		artifact.Kernel == KernelVersion &&
		artifact.Protocol == ProtocolVersion &&
		artifact.Unit == unit.Version &&
		artifact.Optimization == OptimizationVersion &&
		artifact.Enablement == enablement
}

func cacheKey(descriptor rtg.TargetDescriptor, host string) string {
	return cacheKeyForArena(descriptor, host, preparedBackendArenaSize)
}

func cacheKeyForArena(descriptor rtg.TargetDescriptor, host string, arenaSize int) string {
	return cacheKeyForEnablement(descriptor, host, arenaSize, descriptor.Definition)
}

func cacheKeyForEnablement(descriptor rtg.TargetDescriptor, host string, arenaSize int, enablement [32]byte) string {
	identitySource := make([]byte, 0, len(descriptor.Definition)+len(enablement))
	identitySource = append(identitySource, descriptor.Definition[:]...)
	identitySource = append(identitySource, enablement[:]...)
	identity := sha256.Sum256(identitySource)
	return rtg.HashText(identity) + "-" + encodedName(descriptor.Name) +
		"-" + encodedName(host) + "-g" + decimal(rtg.GeneratorVersion) +
		"-k" + decimal(KernelVersion) + "-u" + decimal(unit.Version) +
		"-p" + decimal(ProtocolVersion) + "-o" + decimal(OptimizationVersion) +
		"-a" + decimal(arenaSize) +
		"-c" + backendcompiled.CompilerSourceDigest
}

func encodedName(value string) string {
	const digits = "0123456789abcdef"
	out := make([]byte, len(value)*2)
	for i := 0; i < len(value); i++ {
		out[i*2] = digits[value[i]>>4]
		out[i*2+1] = digits[value[i]&15]
	}
	return string(out)
}

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	var reversed [20]byte
	count := 0
	for value > 0 {
		reversed[count] = byte('0' + value%10)
		count++
		value /= 10
	}
	out := make([]byte, count)
	for i := 0; i < count; i++ {
		out[i] = reversed[count-i-1]
	}
	return string(out)
}

func publish(path string, source []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create backend cache: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".rtgb-*")
	if err != nil {
		return fmt.Errorf("create backend cache entry: %w", err)
	}
	tempPath := temp.Name()
	defer func() { _ = os.Remove(tempPath) }()
	if _, err = temp.Write(source); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write backend cache entry: %w", err)
	}
	if err = temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync backend cache entry: %w", err)
	}
	if err = temp.Close(); err != nil {
		return fmt.Errorf("close backend cache entry: %w", err)
	}
	if err = os.Rename(tempPath, path); err != nil {
		if _, statErr := os.Stat(path); statErr == nil {
			return nil
		}
		return fmt.Errorf("publish backend cache entry: %w", err)
	}
	return nil
}

func prepareFailure(code string, message string) Prepared {
	return Prepared{Diagnostic: driver.Diagnostic{Phase: "backend-prepare", Code: code, Message: message}}
}

func hostTarget() string {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return "linux/amd64"
	case "linux/386":
		return "linux/386"
	case "linux/arm64":
		return "linux/aarch64"
	case "linux/arm":
		return "linux/arm"
	case "windows/amd64":
		return "windows/amd64"
	case "windows/386":
		return "windows/386"
	case "windows/arm64":
		return "windows/arm64"
	case "darwin/arm64":
		return "darwin/arm64"
	case "freebsd/amd64":
		return "freebsd/amd64"
	case "openbsd/amd64":
		return "openbsd/amd64"
	case "netbsd/amd64":
		return "netbsd/amd64"
	case "wasip1/wasm":
		return "wasi/wasm32"
	}
	return ""
}
