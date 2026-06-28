package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"j5.nz/rtg/rtg/build"
	"j5.nz/rtg/rtg/check"
	"j5.nz/rtg/rtg/emit"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/rtgx"
	"j5.nz/rtg/rtg/target"
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
	os.Exit(appMain(os.Args, nil))
}

func appMain(args []string, env []string) int {
	cfg, err := parseArgs(cliArgs(args))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		usage()
		return 1
	}
	if err := run(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cliArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	return args[1:]
}

func run(cfg config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}
	if cfg.link {
		return runLink(cfg)
	}
	graph, err := load.LoadEntries(cfg.inputs, load.Options{Target: cfg.target})
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
	return rtgx.CompileUnits(units, rtgx.Options{Target: cfg.target, Output: cfg.output})
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
	if info, err := os.Stat(cfg.output); err == nil {
		if fileInfoIsDir(info) {
			return writeUnitDirectory(cfg.output, units)
		}
		if len(units) == 1 && isUnitFileOutput(cfg.output) {
			return os.WriteFile(cfg.output, emit.Source(units[0]), 420)
		}
		if len(units) == 1 {
			return fmt.Errorf("rtg: -emit-unit requires .rtg.go output file or output directory")
		}
		return fmt.Errorf("rtg: -emit-unit with multiple packages requires output directory")
	} else if !os.IsNotExist(err) {
		return err
	}
	if len(units) == 1 && isUnitFileOutput(cfg.output) {
		return os.WriteFile(cfg.output, emit.Source(units[0]), 420)
	}
	if filepath.Ext(filepath.Base(cfg.output)) != "" {
		if len(units) == 1 {
			return fmt.Errorf("rtg: -emit-unit requires .rtg.go output file or output directory")
		}
		return fmt.Errorf("rtg: -emit-unit with multiple packages requires output directory")
	}
	return writeUnitDirectory(cfg.output, units)
}

func defaultUnitCacheDir(graph *load.Graph) string {
	return filepath.Join(filepath.Join(graph.Module.Root, ".rtg"), "units")
}

func writeUnitDirectory(dir string, units []unit.Unit) error {
	if err := os.MkdirAll(dir, 493); err != nil {
		return err
	}
	var names []unitFileName
	for i := 0; i < len(units); i++ {
		u := units[i]
		name := emit.FileName(u.ImportPath)
		if existing, ok := findUnitFileName(names, name); ok {
			return fmt.Errorf("rtg: emitted unit filename collision for %s: %s and %s", name, existing, u.ImportPath)
		}
		names = append(names, unitFileName{name: name, importPath: u.ImportPath})
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, emit.Source(u), 420); err != nil {
			return err
		}
	}
	return nil
}

type unitFileName struct {
	name       string
	importPath string
}

func findUnitFileName(names []unitFileName, name string) (string, bool) {
	for i := 0; i < len(names); i++ {
		if names[i].name == name {
			return names[i].importPath, true
		}
	}
	return "", false
}

func isUnitFileOutput(path string) bool {
	return strings.HasSuffix(filepath.Base(path), ".rtg.go")
}

func fileInfoIsDir(info os.FileInfo) bool {
	return info.IsDir()
}

func dirEntryIsDir(entry os.DirEntry) bool {
	return entry.IsDir()
}

func dirEntryName(entry os.DirEntry) string {
	return entry.Name()
}

func runLink(cfg config) error {
	if cfg.output == "" {
		return fmt.Errorf("rtg: -link requires -o")
	}
	if len(cfg.inputs) == 0 {
		return fmt.Errorf("rtg: -link requires input units")
	}
	sources, err := readUnitInputs(cfg.inputs)
	if err != nil {
		return err
	}
	return rtgx.CompileUnitSources(sources, rtgx.Options{Target: cfg.target, Output: cfg.output})
}

func readUnitInputs(inputs []string) ([]unit.SourceFile, error) {
	var sources []unit.SourceFile
	for i := 0; i < len(inputs); i++ {
		input := inputs[i]
		paths, err := unitInputPaths(input)
		if err != nil {
			return nil, err
		}
		for j := 0; j < len(paths); j++ {
			path := paths[j]
			data, err := os.ReadFile(path)
			if err != nil {
				return nil, err
			}
			sources = append(sources, unit.SourceFile{Path: path, Source: data})
		}
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("rtg: no input units")
	}
	return sources, nil
}

func unitInputPaths(input string) ([]string, error) {
	info, err := os.Stat(input)
	if err != nil {
		return nil, err
	}
	if !fileInfoIsDir(info) {
		if !strings.HasSuffix(filepath.Base(input), ".rtg.go") {
			return nil, fmt.Errorf("%s: link input must be an emitted .rtg.go unit", input)
		}
		return []string{input}, nil
	}
	entries, err := os.ReadDir(input)
	if err != nil {
		return nil, err
	}
	var paths []string
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		if dirEntryIsDir(entry) {
			continue
		}
		name := dirEntryName(entry)
		if strings.HasSuffix(name, ".rtg.go") {
			paths = append(paths, filepath.Join(input, name))
		}
	}
	sortStrings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("rtg: no unit files in %s", input)
	}
	return paths, nil
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		value := values[i]
		j := i - 1
		for j >= 0 && stringGreater(values[j], value) {
			values[j+1] = values[j]
			j = j - 1
		}
		values[j+1] = value
	}
}

func stringGreater(a string, b string) bool {
	i := 0
	for i < len(a) && i < len(b) {
		if a[i] > b[i] {
			return true
		}
		if a[i] < b[i] {
			return false
		}
		i = i + 1
	}
	return len(a) > len(b)
}

func parseArgs(args []string) (config, error) {
	cfg := config{target: target.Default()}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-t" {
			i++
			if i >= len(args) {
				return cfg, fmt.Errorf("rtg: missing argument for -t")
			}
			targetArg := args[i]
			if !target.Supported(targetArg) {
				return cfg, fmt.Errorf("rtg: unsupported target: %s\nrtg: supported targets: %s", targetArg, target.List())
			}
			cfg.target = targetArg
			continue
		}
		if arg == "-o" {
			i++
			if i >= len(args) {
				return cfg, fmt.Errorf("rtg: missing argument for -o")
			}
			output := args[i]
			cfg.output = output
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
	if err := validateConfig(cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func validateConfig(cfg config) error {
	modes := 0
	if cfg.check {
		modes++
	}
	if cfg.emitUnit {
		modes++
	}
	if cfg.link {
		modes++
	}
	if modes > 1 {
		return fmt.Errorf("rtg: choose only one of -check, -emit-unit, or -link")
	}
	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: rtg [-t target] [-check] [-emit-unit [-o output.rtg.go|dir]] [-link -o output] [package-or-files...]\nsupported targets: %s\n", target.List())
}
