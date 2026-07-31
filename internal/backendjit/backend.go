//go:build !renvo

package backendjit

import (
	"os"
	"path/filepath"

	"renvo.dev/internal/driver"
	"renvo.dev/internal/linkedimage"
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/rtgb"
	"renvo.dev/internal/runimage"
	"renvo.dev/internal/unit"
)

type Backend struct {
	path        string
	backendRoot string
	stdRoot     string
	cacheDir    string
	bootstrap   driver.Backend
	runner      Runner
	prepared    Prepared
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
	Options  driver.BackendCompileOptions
}

type Runner interface {
	Run(artifact rtgb.Artifact, request Request) driver.BackendResult
}

type ProcessRunner struct{}

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
	inputPath := filepath.Join(tempDir, "input.rtgu")
	outputPath := filepath.Join(tempDir, "output.bin")
	if err = os.WriteFile(inputPath, request.Unit, 0o600); err != nil {
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
	executed := runimage.Run(image, "renvo-prepared-backend", args, os.Environ(), os.Stdin, os.Stdout, os.Stderr)
	if executed.Err != nil || executed.ExitCode != 0 {
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: "RENVO-BACKEND-009", Message: "prepared backend execution failed",
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
