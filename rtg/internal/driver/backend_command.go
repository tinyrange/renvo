//go:build !rtg

package driver

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

type CommandBackend struct {
	Path string
	Args []string
	Env  []string
}

func (b CommandBackend) CompileUnit(unit []byte, target string, strip bool, windowsGUI bool) BackendResult {
	return b.CompileUnitWithArena(unit, target, strip, windowsGUI, 0)
}

func (b CommandBackend) CompileUnitWithArena(unit []byte, target string, strip bool, windowsGUI bool, arenaSize int) BackendResult {
	if b.Path == "" || target == "" || len(unit) == 0 {
		return BackendResult{Diagnostic: Diagnostic{Phase: "backend", Code: "RTG-BACKEND-002", Message: "backend command is not configured"}}
	}
	args := make([]string, 0, len(b.Args)+7)
	args = append(args, b.Args...)
	args = append(args, "-t", target)
	if strip {
		args = append(args, "-s")
	}
	if windowsGUI {
		args = append(args, "-windows-gui")
	}
	if arenaSize > 0 {
		args = append(args, "-arena-size", arenaSizeDecimal(arenaSize))
	}
	args = append(args, "-o", "-", "-")
	cmd := exec.Command(b.Path, args...)
	cmd.Stdin = bytes.NewReader(unit)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if len(b.Env) > 0 {
		cmd.Env = append(os.Environ(), b.Env...)
	}
	err := cmd.Run()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = "backend command failed: " + err.Error()
		}
		return BackendResult{Diagnostic: Diagnostic{Phase: "backend", Code: "RTG-BACKEND-003", Message: message}}
	}
	data := stdout.Bytes()
	if len(data) == 0 {
		return BackendResult{Diagnostic: Diagnostic{Phase: "backend", Code: "RTG-BACKEND-004", Message: "backend produced an empty object"}}
	}
	return BackendResult{Binary: data, Ok: true}
}

func arenaSizeDecimal(value int) string {
	if value == 0 {
		return "0"
	}
	var reversed [10]byte
	count := 0
	for value > 0 {
		reversed[count] = byte('0' + value%10)
		count++
		value = value / 10
	}
	out := make([]byte, count)
	for i := 0; i < count; i++ {
		out[i] = reversed[count-i-1]
	}
	return string(out)
}
