//go:build !renvo

package bootstrap

import (
	"fmt"
	"os"

	"renvo.dev/internal/driver"
)

func Run(args []string, env []string, backend driver.Backend) int {
	args, backend = bootstrapArgs(args, backend)
	args = driver.NormalizeCCompilerCommand(args)
	if driver.TestCommandRequested(args) {
		if len(args) == 3 && (args[2] == "--help" || args[2] == "-h") {
			fmt.Fprint(os.Stdout, driver.TestHelpText)
			return 0
		}
		result := driver.RunTestCommand(args, env, backend, os.Stdin, os.Stdout, os.Stderr)
		if result.Ok {
			return result.ExitCode
		}
		switch result.Error {
		case driver.TestErrCompile:
			printCompileError(result.Compile)
		case driver.TestErrBackend:
			fmt.Fprintln(os.Stderr, "renvo bootstrap test: sibling backend unavailable")
		case driver.TestErrArguments:
			fmt.Fprintf(os.Stderr, "renvo test: invalid arguments: %s\n%s", result.ErrorArg, driver.TestHelpText)
		case driver.TestErrGenerate:
			fmt.Fprintf(os.Stderr, "renvo test: %s\n", result.ErrorArg)
		case driver.TestErrExecute:
			fmt.Fprintf(os.Stderr, "renvo test: execution failed: %s\n", result.ErrorArg)
		default:
			fmt.Fprintln(os.Stderr, "renvo test: failed")
		}
		return result.ExitCode
	}
	if driver.ScriptCommandRequested(args) {
		if len(args) == 3 && (args[2] == "--help" || args[2] == "-h") {
			fmt.Fprint(os.Stdout, driver.RunHelpText)
			return 0
		}
		result := driver.RunScriptCommand(args, env, backend, os.Stdin, os.Stdout, os.Stderr)
		if result.Ok {
			return result.ExitCode
		}
		switch result.Error {
		case driver.RunErrCompile:
			printCompileError(result.Compile)
		case driver.RunErrBackend:
			fmt.Fprintln(os.Stderr, "renvo bootstrap run: sibling backend unavailable")
		case driver.RunErrArguments:
			fmt.Fprintf(os.Stderr, "renvo run: invalid arguments: %s\n%s", result.ErrorArg, driver.RunHelpText)
		case driver.RunErrImage:
			fmt.Fprintln(os.Stderr, "renvo run: backend returned an invalid linked image")
		case driver.RunErrExecute:
			fmt.Fprintf(os.Stderr, "renvo run: execution failed: %s\n", result.ErrorArg)
		default:
			fmt.Fprintln(os.Stderr, "renvo run: failed")
		}
		return result.ExitCode
	}
	if driver.CommandHelpRequested(args) {
		fmt.Fprint(os.Stdout, driver.HelpText)
		return 0
	}
	result := driver.RunCommand(args, env, backend)
	if result.Ok {
		return 0
	}
	printHostError(result)
	return 1
}

func printHostError(result driver.HostResult) {
	switch result.Error {
	case driver.HostErrWorkDir:
		fmt.Fprintln(os.Stderr, "renvo: failed to read working directory")
	case driver.HostErrBackend:
		fmt.Fprintln(os.Stderr, "renvo bootstrap: sibling backend unavailable")
	case driver.HostErrCompile:
		printCompileError(result.Compile)
	case driver.HostErrWrite:
		fmt.Fprintf(os.Stderr, "renvo: failed to write output: %s\n", result.ErrorPath)
	default:
		fmt.Fprintf(os.Stderr, "renvo: failed with host error %d\n", result.Error)
	}
}

func bootstrapArgs(args []string, backend driver.Backend) ([]string, driver.Backend) {
	if len(args) >= 3 && args[1] == "-bootstrap-backend" {
		backend = driver.CommandBackend{Path: args[2]}
		next := make([]string, 1, len(args)-2)
		next[0] = args[0]
		next = append(next, args[3:]...)
		return next, backend
	}
	return args, backend
}

func printCompileError(result driver.CompileResult) {
	if result.Diagnostic.Valid() {
		fmt.Fprint(os.Stderr, driver.FormatDiagnostic(result.Diagnostic))
		return
	}
	switch result.Error {
	case driver.CompileErrBuild:
		printBuildError(result.Build)
	case driver.CompileErrBackend:
		fmt.Fprintln(os.Stderr, "renvo: backend compilation failed")
	default:
		fmt.Fprintf(os.Stderr, "renvo: compilation failed with error %d\n", result.Error)
	}
}

func printBuildError(result driver.BuildResult) {
	if result.Diagnostic.Valid() {
		fmt.Fprint(os.Stderr, driver.FormatDiagnostic(result.Diagnostic))
		return
	}
	switch result.Error {
	case driver.BuildErrOptions:
		printOptionError(result.Options)
	case driver.BuildErrSource:
		fmt.Fprintf(os.Stderr, "renvo: source error at %s\n", result.ErrorPath)
	case driver.BuildErrPipeline:
		fmt.Fprintf(os.Stderr, "renvo: frontend pipeline failed at package=%d file=%d token=%d\n", result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	default:
		fmt.Fprintf(os.Stderr, "renvo: build failed with error %d\n", result.Error)
	}
}

func printOptionError(options driver.Options) {
	switch options.Error {
	case driver.ParseErrMissingOutput:
		fmt.Fprintln(os.Stderr, "renvo: missing output after -o")
	case driver.ParseErrMissingTarget:
		fmt.Fprintln(os.Stderr, "renvo: missing target after -t")
	case driver.ParseErrUnsupportedTarget:
		fmt.Fprintf(os.Stderr, "renvo: unsupported target: %s\n", options.ErrorArg)
	case driver.ParseErrUnknownOption:
		fmt.Fprintf(os.Stderr, "renvo: unknown option: %s\n", options.ErrorArg)
	case driver.ParseErrMissingTags:
		fmt.Fprintln(os.Stderr, "renvo: missing tags after -tags")
	case driver.ParseErrInvalidTags:
		fmt.Fprintf(os.Stderr, "renvo: invalid build tags: %s\n", options.ErrorArg)
	case driver.ParseErrMissingPackage:
		fmt.Fprintln(os.Stderr, "renvo: missing package path")
	case driver.ParseErrExtraPackage:
		fmt.Fprintf(os.Stderr, "renvo: extra package path: %s\n", options.ErrorArg)
	case driver.ParseErrWindowsGUIRequiresWindows:
		fmt.Fprintln(os.Stderr, "renvo: -windows-gui requires a Windows target")
	case driver.ParseErrMixedFileList:
		fmt.Fprintf(os.Stderr, "renvo: explicit source list contains a non-.go argument: %s\n", options.ErrorArg)
	case driver.ParseErrMissingArenaSize:
		fmt.Fprintln(os.Stderr, "renvo: missing arena size after -arena-size")
	case driver.ParseErrInvalidArenaSize:
		fmt.Fprintf(os.Stderr, "renvo: invalid arena size: %s\n", options.ErrorArg)
	case driver.ParseErrScriptRequiresFile:
		fmt.Fprintln(os.Stderr, "renvo: -script requires one explicit .go file")
	case driver.ParseErrScriptFileCount:
		fmt.Fprintln(os.Stderr, "renvo: -script accepts exactly one .go file")
	case driver.ParseErrConflictingEmit:
		fmt.Fprintln(os.Stderr, "renvo: -emit-unit and -emit-image cannot be used together")
	case driver.ParseErrMissingSystem:
		fmt.Fprintln(os.Stderr, "renvo: missing system profile after -system")
	case driver.ParseErrSystemRead, driver.ParseErrInvalidSystem:
		fmt.Fprintf(os.Stderr, "renvo: %s\n", options.SystemError)
	case driver.ParseErrSystemTargetConflict:
		fmt.Fprintln(os.Stderr, "renvo: -system cannot be combined with -t")
	case driver.ParseErrSystemArenaConflict:
		fmt.Fprintln(os.Stderr, "renvo: -system cannot be combined with -arena-size")
	case driver.ParseErrMissingBackend:
		fmt.Fprintln(os.Stderr, "renvo: missing backend after -backend")
	case driver.ParseErrBackendRead:
		fmt.Fprintf(os.Stderr, "renvo: could not read backend: %s\n", options.ErrorArg)
	case driver.ParseErrInvalidBackend:
		fmt.Fprintf(os.Stderr, "renvo: invalid backend: %s\n", options.ErrorArg)
	case driver.ParseErrBackendTarget:
		fmt.Fprintf(os.Stderr, "renvo: backend does not export target: %s\n", options.ErrorArg)
	default:
		fmt.Fprintf(os.Stderr, "renvo: option parse failed with error %d\n", options.Error)
	}
}
