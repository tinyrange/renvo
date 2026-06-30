//go:build !rtg

package rtgx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func compileSourceToBytes(source []byte, target string, backendRootOverride string, stripSymbols bool) ([]byte, error) {
	root, err := backendRoot(backendRootOverride)
	if err != nil {
		return nil, err
	}
	args := []string{"run", ".", "-t", target, "-o", "-"}
	if stripSymbols {
		args = append(args, "-s")
	}
	args = append(args, "-")
	cmd := exec.Command("go", args...)
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

func compileSourceToOutput(source []byte, target string, backendRootOverride string, stripSymbols bool, output string) error {
	data, err := compileSourceToBytes(source, target, backendRootOverride, stripSymbols)
	if err != nil {
		return err
	}
	return writeOutput(data, output)
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
