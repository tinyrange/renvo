package main

import (
	"bytes"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	backend := flag.String("backend", "", "backend source directory")
	output := flag.String("o", "", "generated output")
	sourcesOutput := flag.String("sources", "", "generated embedded source bundle")
	flag.Parse()
	if *backend == "" || *output == "" || *sourcesOutput == "" {
		fmt.Fprintln(os.Stderr, "usage: gen -backend directory -o output.go -sources sources.go")
		os.Exit(2)
	}
	manifest, err := os.ReadFile(filepath.Join(*backend, "compiler_sources.txt"))
	if err != nil {
		fail(err)
	}
	var out bytes.Buffer
	var sourceBundle bytes.Buffer
	var digestSource bytes.Buffer
	out.WriteString("// Code generated from checked-in RTG backend outputs; DO NOT EDIT.\n")
	out.WriteString("package backendcompiled\n\n")
	sourceBundle.WriteString("// Code generated from checked-in RTG backend outputs; DO NOT EDIT.\n")
	sourceBundle.WriteString("package backendcompiled\n\n")
	sourceBundle.WriteString("type CompilerSource struct { Name string; Source string }\n\n")
	sourceBundle.WriteString("var CompilerSources = []CompilerSource{\n")
	for _, line := range strings.Split(string(manifest), "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		source, err := os.ReadFile(filepath.Join(*backend, name))
		if err != nil {
			fail(err)
		}
		packageAt := bytes.Index(source, []byte("package main\n"))
		if packageAt < 0 {
			fail(fmt.Errorf("%s has no package main declaration", name))
		}
		digestSource.WriteString(name)
		digestSource.WriteByte(0)
		digestSource.Write(source)
		out.WriteString("// source: backend/")
		out.WriteString(name)
		out.WriteByte('\n')
		out.Write(source[:packageAt])
		out.Write(source[packageAt+len("package main\n"):])
		out.WriteByte('\n')
		sourceBundle.WriteString("{Name:")
		sourceBundle.WriteString(strconv.Quote(name))
		sourceBundle.WriteString(",Source:")
		sourceBundle.WriteString(strconv.Quote(string(source)))
		sourceBundle.WriteString("},\n")
	}
	sourceBundle.WriteString("}\n")
	digest := fmt.Sprintf("%x", sha256.Sum256(digestSource.Bytes()))
	compilerSource := out.Bytes()
	packageEnd := bytes.Index(compilerSource, []byte("package backendcompiled\n"))
	packageEnd += len("package backendcompiled\n")
	var withDigest bytes.Buffer
	withDigest.Write(compilerSource[:packageEnd])
	withDigest.WriteString("\nconst CompilerSourceDigest = ")
	withDigest.WriteString(strconv.Quote(digest))
	withDigest.WriteString("\n")
	withDigest.Write(compilerSource[packageEnd:])
	compiled := bytes.TrimRight(withDigest.Bytes(), "\n")
	compiled = append(compiled, '\n')
	if err := os.WriteFile(*output, compiled, 0o644); err != nil {
		fail(err)
	}
	if err := os.WriteFile(*sourcesOutput, sourceBundle.Bytes(), 0o644); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "backend bundle:", err)
	os.Exit(1)
}
