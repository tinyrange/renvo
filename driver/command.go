package driver

import (
	"fmt"
	"strings"

	"renvo.dev/internal/backendcompiled"
	internal "renvo.dev/internal/driver"
	"renvo.dev/internal/elflink"
	"renvo.dev/internal/makefile"
)

// CommandRequest compiles a Renvo command using virtual paths only. Args omit
// the executable name; Target and ArenaSize provide defaults for recipes.
type CommandRequest struct {
	Filesystem SourceFS
	Args       []string
	Target     string
	ArenaSize  uint64
	Stdin      []byte
}

// CommandResult retains all generated files in memory, including dependencies.
// Output names the primary file in Outputs; stdout output uses the name "-".
type CommandResult struct {
	Result
	Output  string
	Outputs map[string][]byte
}

func commandResult(ok bool, diagnostic Diagnostic, output string, data []byte, dependency string, dependencies []byte) *CommandResult {
	result := &CommandResult{Result: Result{Ok: ok, Diagnostic: diagnostic, Binary: data}, Output: output, Outputs: make(map[string][]byte)}
	if ok {
		result.Diagnostic = Diagnostic{}
		if output != "" {
			result.Outputs[output] = data
		}
		if dependency != "" && len(dependencies) > 0 {
			result.Outputs[dependency] = dependencies
		}
	}
	return result
}

// CompileCommand runs the portable compiler, preprocessor, or object linker.
// It never launches a process or writes outputs to the host filesystem.
func CompileCommand(request *CommandRequest) (*CommandResult, error) {
	if request == nil || request.Filesystem == nil {
		return nil, fmt.Errorf("source filesystem must be specified")
	}
	args := append([]string{"renvo"}, request.Args...)
	if len(args) == 1 {
		return nil, fmt.Errorf("compiler arguments must be specified")
	}
	fs := &overlayFS{base: request.Filesystem, std: BundledSourceFS()}
	expanded := internal.ExpandCCompilerResponseFiles(args, fs)
	if !expanded.Ok {
		return nil, fmt.Errorf("could not read response file %s", expanded.ErrorPath)
	}
	args = expanded.Args
	if metadata := internal.InspectCCompilerRequest(args); metadata.Kind != internal.CCompilerRequestNone {
		status, output := internal.ExecuteCCompilerRequest(metadata, request.Stdin)
		return commandResult(status == 0, Diagnostic{}, "-", []byte(output), "", nil), nil
	}
	hasTarget, hasArena := false, false
	for _, arg := range args[1:] {
		hasTarget = hasTarget || arg == "-t"
		hasArena = hasArena || arg == "-arena-size"
	}
	if !hasTarget && request.Target != "" {
		args = append(args, "-t", request.Target)
	}
	if isObjectLink(args) {
		return linkCommand(args, fs)
	}
	if !hasArena && request.ArenaSize != 0 {
		args = append(args, "-arena-size", fmt.Sprint(request.ArenaSize))
	}
	args = internal.NormalizeCCompilerCommand(args)
	if internal.CPreprocessCommandRequested(args) {
		r := internal.PreprocessCCommandWithInput(args, ".", fs, request.Stdin)
		return commandResult(r.Ok, internal.CPreprocessCommandDiagnostic(r), r.Output, r.Source, r.DependencyFile, r.DependencyData), nil
	}
	if internal.CAssemblyCommandRequested(args) {
		r := internal.CompileCAssemblyCommand(args, ".", fs)
		return commandResult(r.Ok, internal.CAssemblyCommandDiagnostic(r), r.Output, r.Source, r.DependencyFile, r.DependencyData), nil
	}
	if internal.CSyntaxCommandRequested(args) {
		r := internal.CheckCCommand(args, ".", fs)
		return commandResult(r.Ok, internal.CSyntaxCommandDiagnostic(r), "", nil, r.DependencyFile, r.DependencyData), nil
	}
	r := internal.CompileFromFSWithModuleCache(args[1:], ".", "std/", ".", fs, backendcompiled.Backend{})
	return commandResult(r.Ok, r.Diagnostic, r.Build.Options.Output, r.Binary, r.Build.Options.DependencyFile, internal.CDependencyOutput(r.Build.Options)), nil
}

func isObjectLink(args []string) bool {
	if len(args) < 2 || args[1] != "cc" {
		return false
	}
	object := false
	for _, arg := range args[2:] {
		if arg == "-c" || strings.HasSuffix(arg, ".c") || strings.HasSuffix(arg, ".i") {
			return false
		}
		object = object || strings.HasSuffix(arg, ".o")
	}
	return object
}

func linkCommand(args []string, fs SourceFS) (*CommandResult, error) {
	output := "a.out"
	var inputs []elflink.Input
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if arg == "-o" || arg == "-t" {
			if i+1 == len(args) {
				return nil, fmt.Errorf("%s requires a value", arg)
			}
			i++
			if arg == "-o" {
				output = args[i]
			} else if args[i] != "linux/amd64" {
				return nil, fmt.Errorf("object linking only supports linux/amd64")
			}
			continue
		}
		if arg == "-s" || arg == "-static" || arg == "-nostdlib" {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return nil, fmt.Errorf("unsupported linker option %s", arg)
		}
		data, ok := fs.ReadFile(arg)
		if !ok {
			return nil, fmt.Errorf("could not read object %s", arg)
		}
		inputs = append(inputs, elflink.Input{Name: arg, Data: data})
	}
	image, err := elflink.Link(inputs)
	return commandResult(err.Message == "", Diagnostic{Phase: "linker", Code: "RENVO-LINK-001", Message: err.Message, Path: err.Input}, output, image, "", nil), nil
}

// MakeCommand is one compiler recipe in dependency order.
type MakeCommand = makefile.Command

// PlanMake evaluates Renvo Makefile syntax. exists refers to virtual files;
// recipes are always rebuilt, without requiring host modification times.
func PlanMake(source []byte, targets []string, exists func(string) bool) ([]MakeCommand, error) {
	if exists == nil {
		return nil, fmt.Errorf("virtual file existence callback must be specified")
	}
	file, err := makefile.Parse(source)
	if err.Message != "" {
		return nil, fmt.Errorf("Makefile:%d: %s", err.Line, err.Message)
	}
	// Mark recipe targets phony to rebuild while still checking prerequisites.
	for i := range file.Rules {
		if len(file.Rules[i].Recipes) > 0 {
			file.Rules[i].Phony = true
		}
	}
	commands, err := makefile.Plan(file, targets, func(name string) (int64, bool) { return 0, exists(name) })
	if err.Message != "" {
		return nil, fmt.Errorf("Makefile:%d: %s", err.Line, err.Message)
	}
	return commands, nil
}
