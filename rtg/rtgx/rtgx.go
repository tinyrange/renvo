package rtgx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"j5.nz/rtg/rtg/link"
	targetpkg "j5.nz/rtg/rtg/target"
	"j5.nz/rtg/rtg/unit"
)

type Options struct {
	Target      string
	Output      string
	BackendRoot string
}

type Artifact struct {
	Target             string
	Output             []byte
	LinkedSource       []byte
	LinkedUnits        []string
	ReachableFunctions []string
	Entrypoint         unit.Symbol
}

func CompileUnits(units []unit.Unit, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	artifact, err := CompileUnitsArtifact(units, opts)
	if err != nil {
		return err
	}
	return writeOutput(artifact.Output, opts.Output)
}

func CompileUnitsBytes(units []unit.Unit, opts Options) ([]byte, error) {
	artifact, err := CompileUnitsArtifact(units, opts)
	if err != nil {
		return nil, err
	}
	return artifact.Output, nil
}

func CompileUnitsArtifact(units []unit.Unit, opts Options) (Artifact, error) {
	plan, err := link.Build(units)
	if err != nil {
		return Artifact{}, err
	}
	linked := link.SourceArtifact(plan)
	compiled, err := CompileSourceArtifact(linked.Source, opts)
	if err != nil {
		return Artifact{}, err
	}
	compiled.LinkedSource = append([]byte(nil), linked.Source...)
	compiled.LinkedUnits = append([]string(nil), linked.LinkedUnits...)
	compiled.ReachableFunctions = append([]string(nil), linked.ReachableFunctions...)
	compiled.Entrypoint = linked.Entrypoint
	return compiled, nil
}

func CompileUnitSources(sources []unit.SourceFile, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	artifact, err := CompileUnitSourcesArtifact(sources, opts)
	if err != nil {
		return err
	}
	return writeOutput(artifact.Output, opts.Output)
}

func CompileUnitSourcesBytes(sources []unit.SourceFile, opts Options) ([]byte, error) {
	artifact, err := CompileUnitSourcesArtifact(sources, opts)
	if err != nil {
		return nil, err
	}
	return artifact.Output, nil
}

func CompileUnitSourcesArtifact(sources []unit.SourceFile, opts Options) (Artifact, error) {
	units, err := unit.ParseSources(sources)
	if err != nil {
		return Artifact{}, err
	}
	return CompileUnitsArtifact(units, opts)
}

func CompileSource(source []byte, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	artifact, err := CompileSourceArtifact(source, opts)
	if err != nil {
		return err
	}
	return writeOutput(artifact.Output, opts.Output)
}

func CompileSourceBytes(source []byte, opts Options) ([]byte, error) {
	artifact, err := CompileSourceArtifact(source, opts)
	if err != nil {
		return nil, err
	}
	return artifact.Output, nil
}

func CompileSourceArtifact(source []byte, opts Options) (Artifact, error) {
	target := opts.Target
	if target == "" {
		target = targetpkg.Default()
	}
	if !targetpkg.Supported(target) {
		return Artifact{}, fmt.Errorf("rtg: unsupported target: %s\nrtg: supported targets: %s", target, targetpkg.List())
	}
	data, err := compileSourceToBytes(source, target, opts.BackendRoot)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{Target: target, Output: data}, nil
}

func compileSourceToBytes(source []byte, target string, backendRootOverride string) ([]byte, error) {
	root, err := backendRoot(backendRootOverride)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command("go", "run", ".", "-t", target, "-o", "-", "-")
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(source)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("rtgx compile failed: %w: %s", err, stderr.String())
		}
		return nil, fmt.Errorf("rtgx compile failed: %w", err)
	}
	return stdout.Bytes(), nil
}

func writeOutput(data []byte, outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	if outputPath == "-" {
		_, err := os.Stdout.Write(data)
		return err
	}
	output, err := filepath.Abs(outputPath)
	if err != nil {
		return err
	}
	return os.WriteFile(output, data, 0755)
}

func backendRoot(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if env := os.Getenv("RTGX_ROOT"); env != "" {
		return env, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	root, ok := findBackendRootUpward(cwd)
	if !ok {
		return "", fmt.Errorf("rtg: could not find rtgx backend root; set RTGX_ROOT")
	}
	return root, nil
}

func findBackendRootUpward(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		dir = filepath.Clean(start)
	}
	for {
		if hasBackendRootFiles(dir) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func hasBackendRootFiles(dir string) bool {
	names := []string{"go.mod", "compiler_main.go", "compiler_common_impl.go"}
	for i := 0; i < len(names); i++ {
		name := names[i]
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}
