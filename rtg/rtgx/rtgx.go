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

func CompileUnits(units []unit.Unit, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	data, err := CompileUnitsBytes(units, opts)
	if err != nil {
		return err
	}
	return writeOutput(data, opts.Output)
}

func CompileUnitsBytes(units []unit.Unit, opts Options) ([]byte, error) {
	plan, err := link.Build(units)
	if err != nil {
		return nil, err
	}
	artifact := link.SourceArtifact(plan)
	return CompileSourceBytes(artifact.Source, opts)
}

func CompileUnitSources(sources []unit.SourceFile, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	data, err := CompileUnitSourcesBytes(sources, opts)
	if err != nil {
		return err
	}
	return writeOutput(data, opts.Output)
}

func CompileUnitSourcesBytes(sources []unit.SourceFile, opts Options) ([]byte, error) {
	units, err := unit.ParseSources(sources)
	if err != nil {
		return nil, err
	}
	return CompileUnitsBytes(units, opts)
}

func CompileSource(source []byte, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	data, err := CompileSourceBytes(source, opts)
	if err != nil {
		return err
	}
	return writeOutput(data, opts.Output)
}

func CompileSourceBytes(source []byte, opts Options) ([]byte, error) {
	target := opts.Target
	if target == "" {
		target = targetpkg.Default()
	}
	if !targetpkg.Supported(target) {
		return nil, fmt.Errorf("rtg: unsupported target: %s\nrtg: supported targets: %s", target, targetpkg.List())
	}
	outDir, err := os.MkdirTemp("", "rtgx-out-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(outDir)
	output := filepath.Join(outDir, "out")
	if err := compileSourceToPath(source, target, output, opts.BackendRoot); err != nil {
		return nil, err
	}
	return os.ReadFile(output)
}

func compileSourceToPath(source []byte, target string, output string, backendRootOverride string) error {
	root, err := backendRoot(backendRootOverride)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "run", ".", "-t", target, "-o", output, "-")
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(source)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("rtgx compile failed: %w: %s", err, stderr.String())
		}
		return fmt.Errorf("rtgx compile failed: %w", err)
	}
	return os.Chmod(output, 0755)
}

func writeOutput(data []byte, outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
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
	for _, name := range []string{"go.mod", "compiler_main.go", "compiler_common_impl.go"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}
