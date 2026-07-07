//go:build !rtg

package driver

import (
	"bytes"
	"os"
	"os/exec"
)

type CommandBackend struct {
	Path string
	Args []string
	Env  []string
}

func (b CommandBackend) CompileUnit(unit []byte, target string, strip bool) ([]byte, bool) {
	if b.Path == "" || target == "" || len(unit) == 0 {
		return nil, false
	}
	args := make([]string, 0, len(b.Args)+7)
	args = append(args, b.Args...)
	args = append(args, "-t", target)
	if strip {
		args = append(args, "-s")
	}
	args = append(args, "-o", "-", "-")
	cmd := exec.Command(b.Path, args...)
	cmd.Stdin = bytes.NewReader(unit)
	if len(b.Env) > 0 {
		cmd.Env = append(os.Environ(), b.Env...)
	}
	data, err := cmd.Output()
	if err != nil {
		return nil, false
	}
	if len(data) == 0 {
		return nil, false
	}
	return data, true
}
