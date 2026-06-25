package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"j5.nz/rtg/rtg/build"
	"j5.nz/rtg/rtg/check"
	"j5.nz/rtg/rtg/emit"
	"j5.nz/rtg/rtg/link"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/rtgx"
	"j5.nz/rtg/rtg/unit"
)

type config struct {
	target   string
	output   string
	emitUnit bool
	check    bool
	link     bool
	inputs   []string
}

func main() {
	cfg, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		usage()
		os.Exit(1)
	}
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(cfg config) error {
	if cfg.link {
		return runLink(cfg)
	}
	graph, err := load.LoadEntries(cfg.inputs, load.Options{})
	if err != nil {
		return err
	}
	if cfg.check {
		return check.Graph(graph)
	}
	if cfg.emitUnit {
		return runEmitUnit(cfg, graph)
	}
	return runBuild(cfg, graph)
}

func runBuild(cfg config, graph *load.Graph) error {
	if cfg.output == "" {
		return fmt.Errorf("rtg: build requires -o")
	}
	units, err := build.Units(graph)
	if err != nil {
		return err
	}
	plan, err := link.Build(units)
	if err != nil {
		return err
	}
	return rtgx.CompileSource(link.Source(plan), rtgx.Options{Target: cfg.target, Output: cfg.output})
}

func runEmitUnit(cfg config, graph *load.Graph) error {
	if len(graph.Packages) == 0 {
		return fmt.Errorf("rtg: no packages loaded")
	}
	units, err := build.Units(graph)
	if err != nil {
		return err
	}
	if cfg.output == "" {
		return writeUnitDirectory(defaultUnitCacheDir(graph), units)
	}
	if len(units) == 1 {
		return os.WriteFile(cfg.output, emit.Source(units[0]), 0644)
	}
	if info, err := os.Stat(cfg.output); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("rtg: -emit-unit with multiple packages requires output directory")
		}
	} else if os.IsNotExist(err) {
		if filepath.Ext(cfg.output) == ".go" {
			return fmt.Errorf("rtg: -emit-unit with multiple packages requires output directory")
		}
	} else {
		return err
	}
	return writeUnitDirectory(cfg.output, units)
}

func defaultUnitCacheDir(graph *load.Graph) string {
	return filepath.Join(graph.Module.Root, ".rtg", "units")
}

func writeUnitDirectory(dir string, units []unit.Unit) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	for _, u := range units {
		path := filepath.Join(dir, emit.FileName(u.ImportPath))
		if err := os.WriteFile(path, emit.Source(u), 0644); err != nil {
			return err
		}
	}
	return nil
}

func runLink(cfg config) error {
	if cfg.output == "" {
		return fmt.Errorf("rtg: -link requires -o")
	}
	if len(cfg.inputs) == 0 {
		return fmt.Errorf("rtg: -link requires input units")
	}
	units, err := readUnitInputs(cfg.inputs)
	if err != nil {
		return err
	}
	plan, err := link.Build(units)
	if err != nil {
		return err
	}
	return rtgx.CompileSource(link.Source(plan), rtgx.Options{Target: cfg.target, Output: cfg.output})
}

func readUnitInputs(inputs []string) ([]unit.Unit, error) {
	var units []unit.Unit
	for _, input := range inputs {
		paths, err := unitInputPaths(input)
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			u, err := unit.ParseSource(path, data)
			if err != nil {
				return nil, err
			}
			units = append(units, u)
		}
	}
	if len(units) == 0 {
		return nil, fmt.Errorf("rtg: no input units")
	}
	return units, nil
}

func unitInputPaths(input string) ([]string, error) {
	info, err := os.Stat(input)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{input}, nil
	}
	entries, err := os.ReadDir(input)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".rtg.go") {
			paths = append(paths, filepath.Join(input, name))
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("rtg: no unit files in %s", input)
	}
	return paths, nil
}

func parseArgs(args []string) (config, error) {
	cfg := config{target: "linux/amd64"}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-t" {
			i++
			if i >= len(args) {
				return cfg, fmt.Errorf("rtg: missing argument for -t")
			}
			cfg.target = args[i]
			continue
		}
		if arg == "-o" {
			i++
			if i >= len(args) {
				return cfg, fmt.Errorf("rtg: missing argument for -o")
			}
			cfg.output = args[i]
			continue
		}
		if arg == "-emit-unit" {
			cfg.emitUnit = true
			continue
		}
		if arg == "-check" {
			cfg.check = true
			continue
		}
		if arg == "-link" {
			cfg.link = true
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			return cfg, fmt.Errorf("rtg: unknown option: %s", arg)
		}
		cfg.inputs = append(cfg.inputs, arg)
	}
	if len(cfg.inputs) == 0 && !cfg.link {
		cfg.inputs = append(cfg.inputs, ".")
	}
	return cfg, nil
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: rtg [-t target] [-check] [-emit-unit [-o output.rtg.go|dir]] [-link -o output] [package-or-files...]")
}
