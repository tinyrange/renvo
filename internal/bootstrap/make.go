//go:build !renvo

package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"

	"renvo.dev/internal/driver"
	"renvo.dev/internal/makefile"
)

const makeHelpText = `Usage: renvo make [-f Makefile] [target...]

Evaluates dependency rules and runs recipe lines beginning with renvo. Recipes
are executed directly, without a platform shell, and therefore work unchanged
in the Web IDE virtual filesystem.
`

func makeCommandRequested(args []string) bool {
	return len(args) > 1 && args[1] == "make"
}

func runMakeCommand(args []string, env []string, backend driver.Backend) int {
	path := "Makefile"
	targets := make([]string, 0, 2)
	for i := 2; i < len(args); i++ {
		if args[i] == "-h" || args[i] == "--help" {
			fmt.Fprint(os.Stdout, makeHelpText)
			return 0
		}
		if args[i] == "-f" {
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "renvo make: -f requires a filename")
				return 2
			}
			i++
			path = args[i]
			continue
		}
		if len(args[i]) > 0 && args[i][0] == '-' {
			fmt.Fprintf(os.Stderr, "renvo make: unsupported option: %s\n", args[i])
			return 2
		}
		targets = append(targets, args[i])
	}
	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "renvo make: could not read %s: %v\n", path, err)
		return 1
	}
	file, parseError := makefile.Parse(source)
	if parseError.Message != "" {
		printMakeError(path, parseError)
		return 1
	}
	base := filepath.Dir(path)
	commands, planError := makefile.Plan(file, targets, func(name string) (int64, bool) {
		info, statError := os.Stat(filepath.Join(base, filepath.FromSlash(name)))
		if statError != nil {
			return 0, false
		}
		return info.ModTime().UnixNano(), true
	})
	if planError.Message != "" {
		printMakeError(path, planError)
		return 1
	}
	oldDirectory, directoryError := os.Getwd()
	if directoryError != nil {
		fmt.Fprintln(os.Stderr, "renvo make: could not determine working directory")
		return 1
	}
	if base != "." {
		if err := os.Chdir(base); err != nil {
			fmt.Fprintf(os.Stderr, "renvo make: could not enter %s: %v\n", base, err)
			return 1
		}
		defer os.Chdir(oldDirectory)
	}
	for i := 0; i < len(commands); i++ {
		if !commands[i].Quiet {
			fmt.Fprintln(os.Stdout, commands[i].Text)
		}
		if status := Run(commands[i].Args, env, backend); status != 0 {
			fmt.Fprintf(os.Stderr, "renvo make: target %s failed at %s:%d\n", commands[i].Target, path, commands[i].Line)
			return status
		}
	}
	return 0
}

func printMakeError(path string, err makefile.Error) {
	if err.Line > 0 {
		fmt.Fprintf(os.Stderr, "%s:%d: error RENVO-MAKE-001 (make): %s\n", path, err.Line, err.Message)
	} else {
		fmt.Fprintf(os.Stderr, "%s: error RENVO-MAKE-001 (make): %s\n", path, err.Message)
	}
}
