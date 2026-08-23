package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"renvo.dev/internal/rtg"
)

func main() {
	directory := flag.String("definitions", "", "RTG definition directory")
	output := flag.String("o", "", "generated Go output")
	flag.Parse()
	if *directory == "" || *output == "" {
		fail(fmt.Errorf("usage: gen -definitions directory -o output.go"))
	}
	paths, err := filepath.Glob(filepath.Join(*directory, "*.rtg"))
	if err != nil {
		fail(err)
	}
	sort.Strings(paths)
	sources := make(map[string][]byte, len(paths))
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			fail(err)
		}
		sources[filepath.Base(path)] = source
	}
	loader := sourceLoader{sources: sources}
	roots := make(map[string]string)
	for _, path := range paths {
		name := filepath.Base(path)
		source := sources[name]
		if !bytes.HasPrefix(bytes.TrimSpace(source), []byte("definition ")) {
			continue
		}
		resolved := rtg.ResolveDefinitions(rtg.ParseImports(source, name, loader))
		if !resolved.Ok {
			continue
		}
		for _, target := range resolved.Targets {
			roots[target.Descriptor.Name] = name
			for _, alias := range target.Descriptor.Aliases {
				roots[alias] = name
			}
		}
	}
	var out bytes.Buffer
	out.WriteString("// Code generated from backend/definitions; DO NOT EDIT.\n//go:build !renvo\n\npackage backendbuiltin\n\n")
	out.WriteString("func definitionSource(name string) ([]byte, bool) {\n")
	for _, path := range paths {
		name := filepath.Base(path)
		fmt.Fprintf(&out, "if name == %q { return []byte(%s), true }\n", name, strconv.Quote(string(sources[name])))
	}
	out.WriteString("return nil, false\n}\n\nfunc definitionRoot(target string) (string, bool) {\n")
	var targets []string
	for target := range roots {
		targets = append(targets, target)
	}
	sort.Strings(targets)
	for _, target := range targets {
		fmt.Fprintf(&out, "if target == %q { return %q, true }\n", target, roots[target])
	}
	out.WriteString("return \"\", false\n}\n")
	formatted, err := format.Source(out.Bytes())
	if err != nil {
		fail(fmt.Errorf("format generated catalog: %w", err))
	}
	if err = os.WriteFile(*output, formatted, 0o644); err != nil {
		fail(err)
	}
}

type sourceLoader struct{ sources map[string][]byte }

func (loader sourceLoader) LoadImport(importingFilename string, importPath string) rtg.ImportSource {
	name := filepath.Base(filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath)))
	source, ok := loader.sources[name]
	return rtg.ImportSource{Source: source, Filename: name, Ok: ok}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, strings.TrimSpace(err.Error()))
	os.Exit(1)
}
