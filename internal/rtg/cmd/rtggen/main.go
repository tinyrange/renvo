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
	arch := flag.String("arch", "", "architecture for checked-in architecture generation")
	statefulEmitter := flag.Bool("stateful-emitter", false, "keep the stateful RTG emitter in architecture output")
	kernel := flag.Bool("kernel", false, "generate the shared checked-in architecture kernel")
	packageName := flag.String("package", "backend", "generated Go package")
	output := flag.String("o", "", "generated Go output")
	check := flag.Bool("check", false, "fail if the output is stale")
	flag.Parse()
	if *output == "" || flag.NArg() == 0 && !*kernel {
		fmt.Fprintln(os.Stderr, "usage: rtggen -t target/name -o output.go definition.rtg")
		os.Exit(2)
	}
	if *kernel {
		if *target != "" || *arch != "" || *statefulEmitter || flag.NArg() != 0 {
			fail("kernel generation does not accept definitions, -t, or -arch")
		}
		generated := rtg.GenerateArchitectureKernel(*packageName)
		writeGenerated(*output, *check, generated)
		return
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
	if *arch != "" {
		if *target != "" || len(definitions) != 1 {
			fail("architecture generation requires one definition and no -t")
		}
		if *statefulEmitter {
			generated = rtg.GenerateStatefulArchitectureBackend(definitions[0], *arch, *packageName)
		} else {
			generated = rtg.GenerateArchitectureBackend(definitions[0], *arch, *packageName)
		}
	} else if *target == "" {
		if *statefulEmitter {
			fail("-stateful-emitter requires -arch")
		}
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
	writeGenerated(*output, *check, generated)
}

func writeGenerated(output string, check bool, generated rtg.GenerateResult) {
	if check {
		existing, err := os.ReadFile(output)
		if err != nil || !bytes.Equal(existing, generated.Source) {
			fail(output + " is stale; regenerate it with rtggen")
		}
		return
	}
	if err := os.WriteFile(output, generated.Source, 0o644); err != nil {
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
