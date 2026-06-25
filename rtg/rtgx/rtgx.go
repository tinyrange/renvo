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
	plan, err := link.Build(units)
	if err != nil {
		return err
	}
	return CompileSource(link.Source(plan), opts)
}

func CompileUnitSources(sources []unit.SourceFile, opts Options) error {
	units, err := unit.ParseSources(sources)
	if err != nil {
		return err
	}
	return CompileUnits(units, opts)
}

func CompileSource(source []byte, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	target := opts.Target
	if target == "" {
		target = targetpkg.Default()
	}
	if !targetpkg.Supported(target) {
		return fmt.Errorf("rtg: unsupported target: %s\nrtg: supported targets: %s", target, targetpkg.List())
	}
	output, err := filepath.Abs(opts.Output)
	if err != nil {
		return err
	}
	root, err := backendRoot(opts.BackendRoot)
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
	dir := filepath.Clean(start)
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
