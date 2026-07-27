//go:build !renvo

// Package backendjit prepares external RTG definitions with Renvo's built-in
// host backend and executes the resulting backend image.
package backendjit

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgb"
)

const KernelVersion = 1

type PrepareConfig struct {
	Definition  []byte
	Filename    string
	Target      string
	BackendRoot string
	WorkDir     string
	StdRoot     string
	CacheDir    string
	Bootstrap   driver.Backend
}

type Prepared struct {
	Artifact   rtgb.Artifact
	Encoded    []byte
	CachePath  string
	CacheHit   bool
	Diagnostic driver.Diagnostic
	Ok         bool
}

func Prepare(config PrepareConfig) Prepared {
	resolved := rtg.ResolveDefinitions(rtg.Parse(config.Definition, config.Filename))
	if !resolved.Ok {
		return prepareFailure("RENVO-RTG-001", resolved.Diagnostics[0].Message)
	}
	generated := rtg.GeneratePreparedBackend(resolved, config.Target)
	if !generated.Ok {
		return prepareFailure("RENVO-RTG-002", generated.Diagnostics[0].Message)
	}
	host := hostTarget()
	if host == "" {
		return prepareFailure("RENVO-RTG-003", "this host cannot prepare native backends")
	}
	cachePath := ""
	if config.CacheDir != "" {
		cachePath = filepath.Join(config.CacheDir, cacheKey(generated.Descriptor, host)+".rtgb")
		if source, err := os.ReadFile(cachePath); err == nil {
			if artifact, ok := rtgb.Decode(source); ok && compatible(artifact, generated.Descriptor, host) {
				return Prepared{Artifact: artifact, Encoded: source, CachePath: cachePath, CacheHit: true, Ok: true}
			}
		}
	}
	sources, names, err := preparationSources(config.BackendRoot, generated)
	if err != nil {
		return prepareFailure("RENVO-RTG-004", err.Error())
	}
	args := make([]string, 0, len(names)+7)
	args = append(args, "-s", "-emit-image", "-t", host, "-o", "-")
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
		Descriptor: generated.Descriptor,
		Host:       host,
		Generator:  rtg.GeneratorVersion,
		Kernel:     KernelVersion,
		Payload:    compiled.Binary,
	}
	encoded, ok := rtgb.Encode(artifact)
	if !ok {
		return prepareFailure("RENVO-RTG-006", "prepared backend artifact is invalid")
	}
	if cachePath != "" {
		if err := publish(cachePath, encoded); err != nil {
			return prepareFailure("RENVO-RTG-007", err.Error())
		}
	}
	return Prepared{Artifact: artifact, Encoded: encoded, CachePath: cachePath, Ok: true}
}

func Load(source []byte) Prepared {
	artifact, ok := rtgb.Decode(source)
	if !ok {
		return prepareFailure("RENVO-RTG-008", "invalid prepared backend artifact")
	}
	host := hostTarget()
	if artifact.Host != host || artifact.Generator != rtg.GeneratorVersion ||
		artifact.Kernel != KernelVersion {
		return prepareFailure("RENVO-RTG-009", "prepared backend is incompatible with this compiler or host")
	}
	return Prepared{Artifact: artifact, Encoded: source, Ok: true}
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

func compatible(artifact rtgb.Artifact, descriptor rtg.TargetDescriptor, host string) bool {
	return artifact.Descriptor.Name == descriptor.Name &&
		artifact.Descriptor.Definition == descriptor.Definition &&
		artifact.Descriptor.Version == descriptor.Version &&
		artifact.Host == host &&
		artifact.Generator == rtg.GeneratorVersion &&
		artifact.Kernel == KernelVersion
}

func cacheKey(descriptor rtg.TargetDescriptor, host string) string {
	return rtg.HashText(descriptor.Definition) + "-" + safeName(descriptor.Name) +
		"-" + safeName(host) + "-g" + decimal(rtg.GeneratorVersion) +
		"-k" + decimal(KernelVersion)
}

func safeName(value string) string {
	out := make([]byte, len(value))
	for i := 0; i < len(value); i++ {
		ch := value[i]
		if ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' ||
			ch >= '0' && ch <= '9' || ch == '-' || ch == '_' {
			out[i] = ch
		} else {
			out[i] = '_'
		}
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
	}
	return ""
}
