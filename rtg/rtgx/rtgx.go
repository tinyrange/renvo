package rtgx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Options struct {
	Target      string
	Output      string
	BackendRoot string
}

func CompileSource(source []byte, opts Options) error {
	if opts.Output == "" {
		return fmt.Errorf("rtg: missing output path (-o)")
	}
	target := opts.Target
	if target == "" {
		target = "linux/amd64"
	}
	output, err := filepath.Abs(opts.Output)
	if err != nil {
		return err
	}
	root, err := backendRoot(opts.BackendRoot)
	if err != nil {
		return err
	}
	tmp, err := os.MkdirTemp("", "rtg-rtgx-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	input := filepath.Join(tmp, "linked.rtg.go")
	if err := os.WriteFile(input, source, 0644); err != nil {
		return err
	}
	cmd := exec.Command("go", "run", ".", "-t", target, "-o", output, input)
	cmd.Dir = root
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
