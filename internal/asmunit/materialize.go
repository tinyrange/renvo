// Package asmunit materializes source-preserving assembly without host APIs.
// The caller supplies only a compiler capability for a closed encoder program.
package asmunit

import (
	"renvo.dev/internal/rtg"
	"renvo.dev/internal/unit"
	"renvo.dev/std/vm"
)

// Compiler compiles a self-contained encoder program to raw vm/vm32 bytecode
// The compiler capability may live in a separate arena or process. Execution
// remains inside this service's bounded, host-independent VM.
// No architecture opcodes or pre-materialized frontend bytes are accepted here.
type Compiler interface {
	CompileAssemblyEvaluator(source []byte) ([]byte, string, bool)
}

type Result struct {
	Unit        []byte
	Diagnostics []rtg.Diagnostic
	Ok          bool
}

func Materialize(data []byte, resolved rtg.ResolveResult, target string, compiler Compiler) Result {
	return MaterializeBounded(data, resolved, target, compiler, 96*1024*1024)
}

// MaterializeBounded lets arena-based embedders impose a stricter guest memory
// bound. It cannot raise the service's 96 MiB ceiling or step limit.
func MaterializeBounded(data []byte, resolved rtg.ResolveResult, target string, compiler Compiler, memory int) Result {
	if memory < 256 || memory > 96*1024*1024 {
		return failure("", 0, "invalid assembly evaluator memory bound")
	}
	sources, bindings, ok := unit.ReadRTGAssemblyFragments(data)
	if !ok {
		return failure("", 0, "invalid assembly fragment table")
	}
	if len(bindings) == 0 {
		return Result{Unit: data, Ok: true}
	}
	documents := make([]rtg.AssemblyDocument, len(sources))
	modes := make([]int, len(sources))
	for i := range sources {
		source := sources[i]
		if len(source.Path) > 2 && source.Path[len(source.Path)-2:] == ".s" {
			documents[i] = rtg.LowerAssemblerFile(resolved, target, source.Source, source.Path)
		} else {
			parsed := rtg.ParseAssembly(source.Source, source.Path)
			if parsed.Version == 3 {
				modes[i] = 1
			}
			documents[i] = rtg.LowerTargetAssembly(resolved, target, parsed)
		}
		if !documents[i].Ok {
			return Result{Diagnostics: documents[i].Diagnostics}
		}
	}
	entries := make([]rtg.AssemblyEvaluatorEntry, len(bindings))
	for i := range bindings {
		binding := &bindings[i]
		if binding.Source < 0 || binding.Source >= len(documents) || binding.Entry < 0 || binding.Entry >= len(documents[binding.Source].Entries) {
			return failure("", 0, "invalid assembly entry binding")
		}
		entry := documents[binding.Source].Entries[binding.Entry]
		binding.Mode = modes[binding.Source]
		binding.Inputs = entry.ManagedInputs
		binding.Outputs = entry.ManagedOutputs
		// Always re-evaluate the preserved source, including expert sequence
		// source. This service never trusts Code supplied in a frontend unit.
		entries[i] = rtg.AssemblyEvaluatorEntry{Assembly: documents[binding.Source], EntryIndex: binding.Entry}
	}
	generated := rtg.GenerateAssemblyEvaluators(resolved, target, entries)
	if !generated.Ok {
		return Result{Diagnostics: generated.Diagnostics}
	}
	if compiler == nil {
		return failure("", 0, "assembly evaluator compiler is unavailable")
	}
	bytecode, message, ok := compiler.CompileAssemblyEvaluator(generated.Source)
	if !ok {
		return failure(entries[0].Assembly.Filename, entries[0].Assembly.Entries[entries[0].EntryIndex].Span.Start.Offset, "could not compile assembly evaluator: "+message)
	}
	run := vm.RunConfig(bytecode, vm.Config{Limits: vm.Limits{Steps: 500000000, Memory: memory}})
	if run.Trap != vm.TrapNone || run.ExitCode != 0 {
		return failure("", 0, "assembly evaluator failed")
	}
	output := run.Output
	code := make([][]byte, len(bindings))
	at := 0
	for i := range code {
		if at+4 > len(output) {
			return failure("", 0, "truncated assembly evaluator output")
		}
		size := int(output[at]) | int(output[at+1])<<8 | int(output[at+2])<<16 | int(output[at+3])<<24
		at += 4
		if size <= 0 || size > len(output)-at {
			return failure("", 0, "invalid assembly evaluator frame")
		}
		code[i] = output[at : at+size]
		at += size
	}
	if at != len(output) {
		return failure("", 0, "trailing assembly evaluator output")
	}
	materialized, ok := unit.AttachRTGAssemblyFragments(data, bindings, code)
	if !ok {
		return failure("", 0, "could not attach assembly fragments")
	}
	return Result{Unit: materialized, Ok: true}
}

func failure(path string, offset int, message string) Result {
	return Result{Diagnostics: []rtg.Diagnostic{{Filename: path, Span: rtg.Span{Start: rtg.Position{Offset: offset}}, Code: "RTGASM-MATERIALIZE-001", Message: message}}}
}
