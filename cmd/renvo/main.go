//go:build !renvo

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/backendjit"
	"renvo.dev/internal/bootstrap"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/rtg"
)

func main() {
	env := os.Environ()
	if driver.EnvValue(env, driver.StdRootEnv) == "" {
		stdRoot := "./std"
		if _, err := os.Stat(stdRoot); err != nil {
			stdRoot = "../std"
		}
		env = append(env, driver.StdRootEnv+"="+stdRoot)
	}
	backend := checkoutBackend()
	if len(os.Args) > 2 && os.Args[1] == "backend" && os.Args[2] == "build" {
		os.Exit(buildPreparedBackend(os.Args[3:], env, backend))
	}
	if path := backendPath(os.Args); path != "" {
		backend = backendjit.New(path, checkoutBackendRoot(), driver.StdRootFromEnv(env), backendCacheDir(), backend)
	}
	os.Exit(bootstrap.Run(os.Args, env, backend))
}

func checkoutBackend() driver.Backend {
	return backendcompiled.Backend{}
}

func checkoutBackendRoot() string {
	path := "."
	if _, err := os.Stat("compiler_main.go"); err != nil {
		path = "./backend"
	}
	return path
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

func buildPreparedBackend(args []string, env []string, bootstrapBackend driver.Backend) int {
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
	source, err := os.ReadFile(input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "renvo backend build: could not read definition:", err)
		return 1
	}
	workDir, _ := os.Getwd()
	prepared := backendjit.Prepare(backendjit.PrepareConfig{
		Definition: source, Filename: input, Target: target,
		ImportLoader: commandImportLoader{},
		BackendRoot:  checkoutBackendRoot(), WorkDir: workDir,
		StdRoot: driver.StdRootFromEnv(env), CacheDir: backendCacheDir(),
		Bootstrap: bootstrapBackend,
	})
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

type commandImportLoader struct{}

func (commandImportLoader) LoadImport(
	importingFilename string, importPath string,
) rtg.ImportSource {
	path := filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath))
	imported, err := os.ReadFile(path)
	return rtg.ImportSource{Source: imported, Filename: path, Ok: err == nil}
}
