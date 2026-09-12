// Command renvoemu builds and tests editable .rfe emulator packages.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"renvo.dev/internal/rfe"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "renvoemu:", err)
		os.Exit(1)
	}
}
func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: renvoemu build|test|inspect [-I directory] [-o executable] emulator.rfe")
	}
	mode := args[0]
	f := flag.NewFlagSet("renvoemu "+mode, flag.ContinueOnError)
	search := f.String("I", "", "dependency directory (defaults to source directory)")
	output := f.String("o", "", "output executable")
	root := f.String("renvo-root", "", "Renvo source/runtime checkout (defaults to current checkout)")
	target := f.String("target", "", "Go host target, e.g. windows/amd64 or darwin/arm64")
	testFilter := f.String("run", "", "test name expression")
	bench := f.String("bench", "", "benchmark name expression")
	if err := f.Parse(args[1:]); err != nil {
		return err
	}
	if f.NArg() != 1 {
		return fmt.Errorf("expected one .rfe source")
	}
	if mode != "build" && mode != "test" && mode != "inspect" {
		return fmt.Errorf("unknown command %q", mode)
	}
	source, err := os.ReadFile(f.Arg(0))
	if err != nil {
		return err
	}
	available := map[string][]byte{}
	dir := *search
	if dir == "" {
		dir = filepath.Dir(f.Arg(0))
	}
	var load func([]byte) error
	load = func(data []byte) error {
		p, err := rfe.Decode(data)
		if err != nil {
			return err
		}
		if _, ok := available[p.Manifest.Name]; ok {
			return nil
		}
		available[p.Manifest.Name] = data
		for _, d := range p.Manifest.Requires {
			b, err := os.ReadFile(filepath.Join(dir, d.Name+".rfe"))
			if err != nil {
				return err
			}
			if err = load(b); err != nil {
				return err
			}
		}
		return nil
	}
	if err = load(source); err != nil {
		return err
	}
	packages, err := rfe.Resolve(source, available)
	if err != nil {
		return err
	}
	if mode == "inspect" {
		for _, p := range packages {
			fmt.Printf("%s %s %s\n", p.Manifest.Name, p.Digest, p.Manifest.Import)
		}
		return nil
	}
	if *root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		for {
			if _, err = os.Stat(filepath.Join(cwd, "internal", "rfe", "runtime", "ir.go")); err == nil {
				*root = cwd
				break
			}
			parent := filepath.Dir(cwd)
			if parent == cwd {
				return fmt.Errorf("specify -renvo-root pointing to a Renvo checkout")
			}
			cwd = parent
		}
	}
	*root, err = filepath.Abs(*root)
	if err != nil {
		return err
	}
	workspace, err := os.MkdirTemp("", "renvo-rfe-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workspace)
	if err = rfe.Workspace(workspace, *root, packages); err != nil {
		return err
	}
	command := []string{"test", "-count=1"}
	if *testFilter != "" {
		command = append(command, "-run", *testFilter)
	}
	if *bench != "" {
		command = append(command, "-bench", *bench, "-benchmem")
	}
	command = append(command, "./packages/...")
	if mode == "build" {
		if packages[len(packages)-1].Manifest.Entry == "" {
			return fmt.Errorf("CPU/library RFE has no executable entry")
		}
		if *output == "" {
			return fmt.Errorf("build requires -o")
		}
		*output, err = filepath.Abs(*output)
		if err != nil {
			return err
		}
		command = []string{"build", "-trimpath", "-o", *output, "./cmd"}
	}
	cmd := exec.Command("go", command...)
	cmd.Dir = workspace
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off", "GOSUMDB=off")
	if *target != "" {
		parts := strings.Split(*target, "/")
		if len(parts) != 2 {
			return fmt.Errorf("target must be OS/architecture")
		}
		if mode == "test" {
			return fmt.Errorf("cross-target execution requires a native runner")
		}
		cmd.Env = append(cmd.Env, "GOOS="+parts[0], "GOARCH="+parts[1], "CGO_ENABLED=0")
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
