// Command rtggen is the thin filesystem wrapper around the byte-slice RTG
// definition core. Production preparation uses the same core in memory.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"renvo.dev/internal/rtg"
)

func main() {
	target := flag.String("t", "", "canonical target for fixed generation")
	output := flag.String("o", "", "generated Go output")
	check := flag.Bool("check", false, "fail if the output is stale")
	builtin := flag.Bool("builtin", false, "emit a checked-in built-in backend block without symbol rewriting")
	packageName := flag.String("package", "backend", "generated Go package name")
	goBlock := flag.Int("go-block", 0, "zero-based go backend block for -builtin")
	flag.Parse()
	if flag.NArg() == 0 || *output == "" {
		fmt.Fprintln(os.Stderr, "usage: rtggen -t target/name -o output.go definition.rtg")
		os.Exit(2)
	}
	var definitions []rtg.ResolveResult
	for _, path := range flag.Args() {
		source, err := os.ReadFile(path)
		if err != nil {
			fail(err.Error())
		}
		resolved := rtg.ResolveDefinitions(rtg.Parse(source, path))
		if !resolved.Ok {
			failDiagnostics(resolved.Diagnostics)
		}
		definitions = append(definitions, resolved)
	}
	var generated rtg.GenerateResult
	if *builtin {
		if *target == "" || len(definitions) != 1 {
			fail("built-in generation requires one definition and -t")
		}
		generated = rtg.GenerateBuiltinBackend(definitions[0], *target, *packageName, *goBlock)
	} else if *target == "" {
		generated = rtg.GenerateUniversalBackend(definitions)
	} else {
		if len(definitions) != 1 {
			fail("fixed generation accepts exactly one closed definition")
		}
		generated = rtg.GenerateFixedBackend(definitions[0], *target)
	}
	if !generated.Ok {
		failDiagnostics(generated.Diagnostics)
	}
	if *check {
		existing, err := os.ReadFile(*output)
		if err != nil || !bytes.Equal(existing, generated.Source) {
			fail(*output + " is stale; regenerate it with rtggen")
		}
		return
	}
	if err := os.WriteFile(*output, generated.Source, 0o644); err != nil {
		fail(err.Error())
	}
}

func failDiagnostics(diagnostics []rtg.Diagnostic) {
	for i := 0; i < len(diagnostics); i++ {
		diagnostic := diagnostics[i]
		fmt.Fprintf(os.Stderr, "%s:%d:%d: %s: %s\n", diagnostic.Filename,
			diagnostic.Span.Start.Line, diagnostic.Span.Start.Column,
			diagnostic.Code, diagnostic.Message)
	}
	os.Exit(1)
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "rtggen:", message)
	os.Exit(1)
}
