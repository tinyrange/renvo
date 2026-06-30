package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"j5.nz/rtg/rtg/arena"
	"j5.nz/rtg/rtg/check"
	"j5.nz/rtg/rtg/emit"
	"j5.nz/rtg/rtg/load"
	"j5.nz/rtg/rtg/lower"
	"j5.nz/rtg/rtg/parse"
	"j5.nz/rtg/rtg/rtgx"
	"j5.nz/rtg/rtg/target"
	"j5.nz/rtg/rtg/unit"
)

type config struct {
	target   string
	output   string
	strip    bool
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
		printError(err)
		usage()
		return 1
	}
	err = run(cfg)
	if err != nil {
		printError(err)
		return 1
	}
	return 0
}

func printError(err error) {
	fmt.Println(err.Error())
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
	loadMark := arena.Mark()
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
	return runBuild(cfg, graph, loadMark)
}

func runBuild(cfg config, graph *load.Graph, loadMark int) error {
	if cfg.output == "" {
		return fmt.Errorf("rtg: build requires -o")
	}
	checkMark := arena.Mark()
	if err := check.Graph(graph); err != nil {
		arena.Reset(checkMark)
		return err
	}
	arena.Reset(checkMark)
	clearGraphParsedFiles(graph)
	unitDir := cfg.output + ".units"
	persistMark := arena.PersistMark()
	if err := writeGraphUnitDirectory(unitDir, graph); err != nil {
		arena.PersistReset(persistMark)
		return err
	}
	arena.PersistReset(persistMark)
	unitDir = arena.PersistString(unitDir)
	target := arena.PersistString(cfg.target)
	output := arena.PersistString(cfg.output)
	strip := cfg.strip
	arena.Reset(loadMark)
	unitSourceMark := arena.Mark()
	sources, err := readUnitTextInputs([]string{unitDir})
	if err != nil {
		arena.Reset(unitSourceMark)
		return err
	}
	return rtgx.CompileTextUnitSourcesReset(sources, rtgx.Options{Target: target, Output: output, StripSymbols: strip}, unitSourceMark)
}

func clearGraphParsedFiles(graph *load.Graph) {
	for i := 0; i < len(graph.Packages); i++ {
		files := graph.Packages[i].Files
		for j := 0; j < len(files); j++ {
			files[j].Parsed = parse.File{}
		}
		graph.Packages[i].Files = files
	}
}

func runEmitUnit(cfg config, graph *load.Graph) error {
	if len(graph.Packages) == 0 {
		return fmt.Errorf("rtg: no packages loaded")
	}
	if cfg.output == "" {
		dir := defaultUnitCacheDir(graph)
		return writeGraphUnitDirectory(dir, graph)
	}
	info, statErr := os.Stat(cfg.output)
	if statErr == nil {
		if fileInfoIsDir(info) {
			return writeGraphUnitDirectory(cfg.output, graph)
		}
		if len(graph.Packages) == 1 && isUnitFileOutput(cfg.output) {
			return writeSingleGraphUnitFile(cfg.output, graph)
		}
		if len(graph.Packages) == 1 {
			return fmt.Errorf("rtg: -emit-unit requires .rtg.go output file or output directory")
		}
		return fmt.Errorf("rtg: -emit-unit with multiple packages requires output directory")
	}
	if !os.IsNotExist(statErr) {
		return statErr
	}
	if len(graph.Packages) == 1 && isUnitFileOutput(cfg.output) {
		return writeSingleGraphUnitFile(cfg.output, graph)
	}
	base := filepath.Base(cfg.output)
	ext := filepath.Ext(base)
	if ext != "" {
		if len(graph.Packages) == 1 {
			return fmt.Errorf("rtg: -emit-unit requires .rtg.go output file or output directory")
		}
		return fmt.Errorf("rtg: -emit-unit with multiple packages requires output directory")
	}
	return writeGraphUnitDirectory(cfg.output, graph)
}

func defaultUnitCacheDir(graph *load.Graph) string {
	return filepath.Join(filepath.Join(graph.Module.Root, ".rtg"), "units")
}

func writeUnitDirectory(dir string, units []unit.Unit) error {
	err := os.MkdirAll(dir, 493)
	if err != nil {
		return err
	}
	names := make([]unitFileName, 0, len(units))
	for i := 0; i < len(units); i++ {
		u := units[i]
		name := emit.FileName(u.ImportPath)
		if existing, ok := findUnitFileName(names, name); ok {
			return fmt.Errorf("rtg: emitted unit filename collision for %s: %s and %s", name, existing, u.ImportPath)
		}
		names = append(names, unitFileName{name: name, importPath: u.ImportPath})
		path := filepath.Join(dir, name)
		data := emit.Source(u)
		err = os.WriteFile(path, data, 420)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeGraphUnitDirectory(dir string, graph *load.Graph) error {
	err := os.MkdirAll(dir, 493)
	if err != nil {
		return err
	}
	names := make([]unitFileName, 0, len(graph.Packages))
	for i := 0; i < len(graph.Packages); i++ {
		pkg := graph.Packages[i]
		name := emit.FileName(pkg.ImportPath)
		if existing, ok := findUnitFileName(names, name); ok {
			return fmt.Errorf("rtg: emitted unit filename collision for %s: %s and %s", name, existing, pkg.ImportPath)
		}
		names = append(names, unitFileName{name: name, importPath: pkg.ImportPath})
		mark := arena.Mark()
		u, err := lower.PackageWithGraph(pkg, graph)
		if err != nil {
			arena.Reset(mark)
			return err
		}
		path := filepath.Join(dir, name)
		data := emit.Source(u)
		err = os.WriteFile(path, data, 420)
		if err != nil {
			arena.Reset(mark)
			return err
		}
		arena.Reset(mark)
	}
	return nil
}

func writeSingleGraphUnitFile(path string, graph *load.Graph) error {
	u, err := lower.PackageWithGraph(graph.Packages[0], graph)
	if err != nil {
		return err
	}
	data := emit.Source(u)
	return os.WriteFile(path, data, 420)
}

func graphUnitInputPaths(dir string, graph *load.Graph) []string {
	paths := make([]string, 0, len(graph.Packages))
	for i := 0; i < len(graph.Packages); i++ {
		name := emit.FileName(graph.Packages[i].ImportPath)
		paths = append(paths, filepath.Join(dir, name))
	}
	sortStrings(paths)
	return paths
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
	unitSourceMark := arena.Mark()
	sources, err := readUnitTextInputs(cfg.inputs)
	if err != nil {
		arena.Reset(unitSourceMark)
		return err
	}
	target := arena.PersistString(cfg.target)
	output := arena.PersistString(cfg.output)
	strip := cfg.strip
	return rtgx.CompileTextUnitSourcesReset(sources, rtgx.Options{Target: target, Output: output, StripSymbols: strip}, unitSourceMark)
}

func readUnitTextInputs(inputs []string) ([]unit.TextSourceFile, error) {
	var paths []string
	for i := 0; i < len(inputs); i++ {
		input := inputs[i]
		inputPaths, err := unitInputPaths(input)
		if err != nil {
			return nil, err
		}
		paths = appendStrings(paths, inputPaths)
	}
	return readUnitTextPaths(paths)
}

func readUnitTextPaths(paths []string) ([]unit.TextSourceFile, error) {
	var sources []unit.TextSourceFile
	for i := 0; i < len(paths); i++ {
		path := paths[i]
		text, err := readUnitTextFile(path)
		if err != nil {
			return nil, err
		}
		sources = append(sources, unit.TextSourceFile{Path: path, Source: text})
	}
	if len(sources) == 0 {
		return nil, fmt.Errorf("rtg: no input units")
	}
	return sources, nil
}

func readUnitInputs(inputs []string) ([]unit.SourceFile, error) {
	var paths []string
	for i := 0; i < len(inputs); i++ {
		input := inputs[i]
		inputPaths, err := unitInputPaths(input)
		if err != nil {
			return nil, err
		}
		paths = appendStrings(paths, inputPaths)
	}
	return readUnitPaths(paths)
}

func readUnitPaths(paths []string) ([]unit.SourceFile, error) {
	var sources []unit.SourceFile
	for i := 0; i < len(paths); i++ {
		path := paths[i]
		data, err := readUnitFile(path)
		if err != nil {
			return nil, err
		}
		sources = append(sources, unit.SourceFile{Path: path, Source: data})
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

func appendStrings(out []string, values []string) []string {
	for i := 0; i < len(values); i++ {
		out = append(out, values[i])
	}
	return out
}

func parseArgs(args []string) (config, error) {
	cfg := config{target: target.Default()}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-t" {
			i++
			if i >= len(args) {
				err := fmt.Errorf("rtg: missing argument for -t")
				return cfg, err
			}
			targetArg := args[i]
			if !target.Supported(targetArg) {
				err := fmt.Errorf("rtg: unsupported target: %s\nrtg: supported targets: %s", targetArg, target.List())
				return cfg, err
			}
			cfg.target = targetArg
			continue
		}
		if arg == "-o" {
			i++
			if i >= len(args) {
				err := fmt.Errorf("rtg: missing argument for -o")
				return cfg, err
			}
			output := args[i]
			cfg.output = output
			continue
		}
		if arg == "-s" {
			cfg.strip = true
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
			err := fmt.Errorf("rtg: unknown option: %s", arg)
			return cfg, err
		}
		cfg.inputs = append(cfg.inputs, arg)
	}
	if len(cfg.inputs) == 0 && !cfg.link {
		cfg.inputs = append(cfg.inputs, ".")
	}
	err := validateConfig(cfg)
	if err != nil {
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
}
