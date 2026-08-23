//go:build !renvo

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"renvo.dev/internal/backendjit"
	"renvo.dev/internal/backendvm32"
	"renvo.dev/internal/bootstrap"
	"renvo.dev/internal/driver"
)

func main() {
	env := os.Environ()
	if driver.EnvValue(env, driver.StdRootEnv) == "" {
		stdRoot := "./std"
		if _, err := os.Stat(stdRoot); err != nil {
			stdRoot = "../std"
		}
		if _, err := os.Stat(stdRoot); err == nil {
			env = append(env, driver.StdRootEnv+"="+stdRoot)
		}
	}
	stdRoot := driver.StdRootFromEnv(env)
	backend := checkoutBackend(stdRoot, backendCacheDir())
	if len(os.Args) > 2 && os.Args[1] == "backend" && os.Args[2] == "build" {
		os.Exit(buildPreparedBackend(os.Args[3:], env))
	}
	if path := backendPath(os.Args); path != "" {
		backend = backendjit.NewSeeded(path, stdRoot, backendCacheDir())
	}
	os.Exit(bootstrap.Run(os.Args, env, backend))
}

// checkoutBackendMux keeps direct VM32 output on the fixed seed and delegates
// other built-in targets to the backend selected by the Go build variant.
type checkoutBackendMux struct {
	builtin checkoutBuiltinBackend
}

type checkoutBuiltinBackend interface {
	driver.Backend
	driver.ArenaBackend
	driver.OptionsBackend
}

func (b checkoutBackendMux) CompileUnit(unit []byte, target string, strip bool, windowsGUI bool) driver.BackendResult {
	if target == backendvm32.Target {
		return (backendvm32.Backend{}).CompileUnit(unit, target, strip, windowsGUI)
	}
	return b.builtin.CompileUnit(unit, target, strip, windowsGUI)
}

func (b checkoutBackendMux) CompileUnitWithArena(unit []byte, target string, strip bool, windowsGUI bool, arenaSize int) driver.BackendResult {
	if target == backendvm32.Target {
		return (backendvm32.Backend{}).CompileUnitWithArena(unit, target, strip, windowsGUI, arenaSize)
	}
	return b.builtin.CompileUnitWithArena(unit, target, strip, windowsGUI, arenaSize)
}

func (b checkoutBackendMux) CompileUnitWithOptions(unit []byte, options driver.BackendCompileOptions) driver.BackendResult {
	if options.Target == backendvm32.Target {
		return (backendvm32.Backend{}).CompileUnitWithOptions(unit, options)
	}
	return b.builtin.CompileUnitWithOptions(unit, options)
}

func backendPath(args []string) string {
	for i := 1; i+1 < len(args); i++ {
		if args[i] == "-backend" {
			return args[i+1]
		}
	}
	return ""
}

func backendCacheDir() string {
	root, err := os.UserCacheDir()
	if err != nil || root == "" {
		return ""
	}
	return filepath.Join(root, "renvo", "backends")
}

func buildPreparedBackend(args []string, env []string) int {
	input := ""
	target := ""
	output := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-t":
			i++
			if i < len(args) {
				target = args[i]
			}
		case "-o":
			i++
			if i < len(args) {
				output = args[i]
			}
		default:
			if input == "" {
				input = args[i]
			} else {
				fmt.Fprintln(os.Stderr, "renvo backend build: unexpected argument:", args[i])
				return 2
			}
		}
	}
	if input == "" || target == "" || output == "" {
		fmt.Fprintln(os.Stderr, "usage: renvo backend build <definition.rtg> -t <target> -o <backend.rtgb>")
		return 2
	}
	_, err := os.ReadFile(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "renvo backend build: could not read definition:", err)
		return 1
	}
	prepared := backendjit.NewSeeded(input, driver.StdRootFromEnv(env), backendCacheDir()).Prepare(target)
	if !prepared.Ok {
		fmt.Fprint(os.Stderr, driver.FormatDiagnostic(prepared.Diagnostic))
		return 1
	}
	if err := os.WriteFile(output, prepared.Encoded, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "renvo backend build: could not write artifact:", err)
		return 1
	}
	return 0
}
