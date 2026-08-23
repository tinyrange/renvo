//go:build !renvo

// Command renvowasivmbackend executes a prepared VM32 backend compiler inside
// a WASI process and publishes the compiler's output into the WASI filesystem.
package main

import (
	"fmt"
	"os"

	"renvo.dev/std/vm"
)

func main() {
	if len(os.Args) < 4 || os.Args[2] != "--" {
		fail("usage: renvo-vm-backend compiler.rnvb -- [compiler arguments]")
	}
	program, err := os.ReadFile(os.Args[1])
	if err != nil {
		fail("read compiler: " + err.Error())
	}
	args := os.Args[3:]
	input := inputName(args)
	if input == "" {
		fail("compiler input is missing")
	}
	data, err := os.ReadFile(input)
	if err != nil {
		fail("read compiler input: " + err.Error())
	}
	compilerArgs := stdinArguments(args, input)
	result := vm.RunConfig(program, vm.Config{
		Limits: vm.Limits{Steps: 2_000_000_000, Memory: 128 * 1024 * 1024},
		Args:   append([]string{"renvo-prepared-backend"}, compilerArgs...),
		Env:    []string{"PWD=/"}, Stdin: data,
	})
	_, _ = os.Stdout.Write(result.Output)
	_, _ = os.Stderr.Write(result.Stderr)
	if result.Trap != vm.TrapNone {
		fail(fmt.Sprintf("VM trap %d at pc %d (opcode %d, %d steps)",
			result.Trap, result.TrapPC, result.TrapOpcode, result.Steps))
	}
	if result.ExitCode != 0 {
		os.Exit(result.ExitCode)
	}
	for _, file := range result.Files {
		if file.Name == input {
			continue
		}
		if err = os.WriteFile(file.Name, file.Data, os.FileMode(file.Mode)); err != nil {
			fail("write compiler output: " + err.Error())
		}
	}
}

func stdinArguments(args []string, input string) []string {
	result := append([]string(nil), args...)
	for i := len(result) - 1; i >= 0; i-- {
		if result[i] == input && (i == 0 || !optionTakesValue(result[i-1])) {
			result[i] = "-"
			break
		}
	}
	return result
}

func inputName(args []string) string {
	for i := len(args) - 1; i >= 0; i-- {
		if args[i] != "" && args[i][0] != '-' {
			if i > 0 && optionTakesValue(args[i-1]) {
				continue
			}
			return args[i]
		}
	}
	return ""
}

func optionTakesValue(option string) bool {
	return option == "-t" || option == "-o" || option == "-arena-size" ||
		option == "-module-name" || option == "-module-license"
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "renvo-vm-backend:", message)
	os.Exit(1)
}
