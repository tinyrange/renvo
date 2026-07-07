//go:build !rtg

package driver

import (
	"io"
	"os"
	"strings"
	"testing"

	"j5.nz/rtg/rtg/internal/unit"
)

func TestCommandBackendCompileUnit(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable failed: %v", err)
	}
	built := BuildUnit([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles())
	if !built.Ok {
		t.Fatalf("BuildUnit failed: %#v", built)
	}

	backend := CommandBackend{
		Path: exe,
		Args: []string{"-test.run=TestCommandBackendHelper", "--"},
		Env:  []string{"RTG_DRIVER_COMMAND_BACKEND_HELPER=1"},
	}
	binary, ok := backend.CompileUnit(built.Unit, "linux/386", true)
	if !ok {
		t.Fatal("CommandBackend failed")
	}
	if string(binary) != "compiled linux/386 strip\n" {
		t.Fatalf("binary = %q", string(binary))
	}
}

func TestCommandBackendFailure(t *testing.T) {
	if _, ok := (CommandBackend{}).CompileUnit([]byte("unit"), "linux/amd64", false); ok {
		t.Fatal("empty backend path was accepted")
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable failed: %v", err)
	}
	built := BuildUnit([]string{"-o", "app", "./cmd/app"}, "/repo/case", "/std", driverTestFiles())
	if !built.Ok {
		t.Fatalf("BuildUnit failed: %#v", built)
	}
	backend := CommandBackend{
		Path: exe,
		Args: []string{"-test.run=TestCommandBackendHelper", "--"},
		Env:  []string{"RTG_DRIVER_COMMAND_BACKEND_HELPER=1", "RTG_DRIVER_COMMAND_BACKEND_FAIL=1"},
	}
	if _, ok := backend.CompileUnit(built.Unit, "linux/amd64", false); ok {
		t.Fatal("failing backend was accepted")
	}
}

func TestCommandBackendHelper(t *testing.T) {
	if os.Getenv("RTG_DRIVER_COMMAND_BACKEND_HELPER") != "1" {
		return
	}
	if os.Getenv("RTG_DRIVER_COMMAND_BACKEND_FAIL") == "1" {
		os.Exit(2)
	}
	args := helperBackendArgs(os.Args)
	target, output, input, strip, ok := parseHelperBackendArgs(args)
	if !ok {
		os.Exit(3)
	}
	if input != "-" || output != "-" {
		os.Exit(7)
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(4)
	}
	program, unitOk := unit.Unmarshal(data)
	if !unitOk || program.Package != "main" {
		os.Exit(5)
	}
	text := "compiled " + target
	if strip {
		text = text + " strip"
	}
	text = text + "\n"
	if _, err := os.Stdout.Write([]byte(text)); err != nil {
		os.Exit(6)
	}
	os.Exit(0)
}

func helperBackendArgs(args []string) []string {
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			return args[i+1:]
		}
	}
	return nil
}

func parseHelperBackendArgs(args []string) (string, string, string, bool, bool) {
	target := ""
	output := ""
	input := ""
	strip := false
	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "-s" {
			strip = true
			i++
			continue
		}
		if arg == "-t" {
			if i+1 >= len(args) {
				return "", "", "", false, false
			}
			target = args[i+1]
			i += 2
			continue
		}
		if arg == "-o" {
			if i+1 >= len(args) {
				return "", "", "", false, false
			}
			output = args[i+1]
			i += 2
			continue
		}
		if input != "" {
			return "", "", "", false, false
		}
		if arg != "-" && strings.HasPrefix(arg, "-") {
			return "", "", "", false, false
		}
		input = arg
		i++
	}
	return target, output, input, strip, target != "" && output != "" && input != ""
}
