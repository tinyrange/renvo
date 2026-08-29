// Command rtggen is the thin filesystem wrapper around the byte-slice RTG
// definition core. Production preparation uses the same core in memory.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/build/constraint"
	"os"
	"path/filepath"

	"renvo.dev/internal/rbe"
	"renvo.dev/internal/rtg"
)

func main() {
	target := flag.String("t", "", "canonical target for fixed generation")
	arch := flag.String("arch", "", "architecture for checked-in architecture generation")
	statefulEmitter := flag.Bool("stateful-emitter", false, "keep the stateful RTG emitter in architecture output")
	algorithms := flag.Bool("algorithms", false, "emit the pruned checked-in algorithm projection")
	compilerIntegration := flag.Bool("compiler-integration", false, "emit RTG-owned compiler-private lowering")
	contract := flag.Bool("contract", false, "emit the checked-in semantic contract projection")
	targetProjection := flag.Bool("target-projection", false, "emit a checked-in production target projection")
	prepared := flag.Bool("prepared", false, "emit a prepared custom backend for compiler package main")
	kernel := flag.Bool("kernel", false, "generate the shared checked-in architecture kernel")
	packageName := flag.String("package", "backend", "generated Go package")
	buildTag := flag.String("build-tag", "", "prepend one Go build tag to generated source")
	output := flag.String("o", "", "generated Go output")
	check := flag.Bool("check", false, "fail if the output is stale")
	flag.Parse()
	if *output == "" || flag.NArg() == 0 && !*kernel {
		fmt.Fprintln(os.Stderr, "usage: rtggen -t target/name -o output.go definition.rtg")
		os.Exit(2)
	}
	if *kernel {
		if *target != "" || *arch != "" || *statefulEmitter || *compilerIntegration ||
			*targetProjection || flag.NArg() != 0 {
			fail("kernel generation does not accept definitions, -t, or -arch")
		}
		generated := rtg.GenerateArchitectureKernel(*packageName)
		writeGenerated(*output, *check, generated)
		return
	}
	if *compilerIntegration && *arch == "" {
		fail("compiler integration generation requires -arch")
	}
	var definitions []rtg.ResolveResult
	for _, path := range flag.Args() {
		source, err := os.ReadFile(path)
		if err != nil {
			fail(err.Error())
		}
		bundle := rbe.Parse(source)
		if !bundle.Ok {
			fail(fmt.Sprintf("%s:%d: %s", path, bundle.Offset, bundle.Message))
		}
		source = bundle.Definition
		parsed := rtg.ParseImports(source, path, filesystemImportLoader{})
		var resolved rtg.ResolveResult
		if *arch != "" {
			resolved = rtg.ResolveArchitectureDefinition(parsed)
		} else {
			resolved = rtg.ResolveDefinitions(parsed)
		}
		if !resolved.Ok {
			failDiagnostics(resolved.Diagnostics)
		}
		definitions = append(definitions, resolved)
	}
	var generated rtg.GenerateResult
	if *arch != "" {
		if *target != "" || *prepared || *targetProjection || len(definitions) != 1 {
			fail("architecture generation requires one definition and no -t")
		}
		modeCount := 0
		for _, enabled := range []bool{*algorithms, *compilerIntegration, *contract, *statefulEmitter} {
			if enabled {
				modeCount++
			}
		}
		if modeCount > 1 {
			fail("architecture generation modes are mutually exclusive")
		}
		if *algorithms {
			generated = rtg.GenerateCheckedInArchitectureAlgorithms(definitions[0], *arch, *packageName)
		} else if *compilerIntegration {
			generated = rtg.GenerateCheckedInCompilerIntegration(definitions[0], *arch, *packageName)
		} else if *contract {
			generated = rtg.GenerateCheckedInArchitectureContract(definitions[0], *arch, *packageName)
		} else if *statefulEmitter {
			generated = rtg.GenerateStatefulArchitectureBackend(definitions[0], *arch, *packageName)
		} else {
			generated = rtg.GenerateArchitectureBackend(definitions[0], *arch, *packageName)
		}
	} else if *targetProjection {
		if *target == "" || *prepared || *statefulEmitter || *algorithms || *compilerIntegration || *contract ||
			len(definitions) != 1 {
			fail("target projection requires one definition, -t, and no other generation mode")
		}
		generated = rtg.GenerateCheckedInTargetProjection(definitions[0], *target, *packageName)
	} else if *target == "" {
		if *statefulEmitter {
			fail("-stateful-emitter requires -arch")
		}
		generated = rtg.GenerateUniversalBackend(definitions)
	} else if *prepared {
		if len(definitions) != 1 {
			fail("prepared generation accepts exactly one closed definition")
		}
		generated = rtg.GeneratePreparedBackend(definitions[0], *target)
	} else {
		if len(definitions) != 1 {
			fail("fixed generation accepts exactly one closed definition")
		}
		generated = rtg.GenerateFixedBackend(definitions[0], *target)
	}
	if !generated.Ok {
		failDiagnostics(generated.Diagnostics)
	}
	if *buildTag != "" {
		if !validBuildTag(*buildTag) {
			fail("-build-tag must be one Go identifier")
		}
		prefix := []byte("//go:build " + *buildTag + "\n\n")
		generated.Source = append(prefix, generated.Source...)
	}
	writeGenerated(*output, *check, generated)
}

func validBuildTag(tag string) bool {
	expression, err := constraint.Parse("//go:build " + tag)
	if err != nil {
		return false
	}
	_, ok := expression.(*constraint.TagExpr)
	return ok
}

type filesystemImportLoader struct{}

func (filesystemImportLoader) LoadImport(
	importingFilename string, importPath string,
) rtg.ImportSource {
	path := filepath.Clean(filepath.Join(filepath.Dir(importingFilename), importPath))
	source, err := os.ReadFile(path)
	return rtg.ImportSource{Source: source, Filename: path, Ok: err == nil}
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
