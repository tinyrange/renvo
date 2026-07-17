//go:build !rtg

package main

import (
	"fmt"
	"os"

	"j5.nz/rtg/rtg/internal/driver"
)

func main() {
	os.Exit(run(os.Args, os.Environ()))
}

func run(args []string, env []string) int {
	if driver.CommandHelpRequested(args) {
		fmt.Fprint(os.Stdout, driver.HelpText)
		return 0
	}
	result := driver.RunCommand(args, env, nil)
	if result.Ok {
		return 0
	}
	printHostError(result)
	return 1
}

func printHostError(result driver.HostResult) {
	switch result.Error {
	case driver.HostErrWorkDir:
		fmt.Fprintln(os.Stderr, "rtg: failed to read working directory")
	case driver.HostErrBackend:
		fmt.Fprintf(os.Stderr, "rtg: backend unavailable; set %s\n", driver.BackendEnv)
	case driver.HostErrCompile:
		printCompileError(result.Compile)
	case driver.HostErrWrite:
		fmt.Fprintf(os.Stderr, "rtg: failed to write output: %s\n", result.ErrorPath)
	default:
		fmt.Fprintf(os.Stderr, "rtg: failed with host error %d\n", result.Error)
	}
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
		fmt.Fprintln(os.Stderr, "rtg: backend compilation failed")
	default:
		fmt.Fprintf(os.Stderr, "rtg: compilation failed with error %d\n", result.Error)
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
		fmt.Fprintf(os.Stderr, "rtg: source error at %s\n", result.ErrorPath)
	case driver.BuildErrPipeline:
		fmt.Fprintf(os.Stderr, "rtg: frontend pipeline failed at package=%d file=%d token=%d\n", result.ErrorPackage, result.ErrorFile, result.ErrorToken)
	default:
		fmt.Fprintf(os.Stderr, "rtg: build failed with error %d\n", result.Error)
	}
}

func printOptionError(options driver.Options) {
	switch options.Error {
	case driver.ParseErrMissingOutput:
		fmt.Fprintln(os.Stderr, "rtg: missing output after -o")
	case driver.ParseErrMissingTarget:
		fmt.Fprintln(os.Stderr, "rtg: missing target after -t")
	case driver.ParseErrUnsupportedTarget:
		fmt.Fprintf(os.Stderr, "rtg: unsupported target: %s\n", options.ErrorArg)
	case driver.ParseErrUnknownOption:
		fmt.Fprintf(os.Stderr, "rtg: unknown option: %s\n", options.ErrorArg)
	case driver.ParseErrMissingTags:
		fmt.Fprintln(os.Stderr, "rtg: missing tags after -tags")
	case driver.ParseErrInvalidTags:
		fmt.Fprintf(os.Stderr, "rtg: invalid build tags: %s\n", options.ErrorArg)
	case driver.ParseErrMissingPackage:
		fmt.Fprintln(os.Stderr, "rtg: missing package path")
	case driver.ParseErrExtraPackage:
		fmt.Fprintf(os.Stderr, "rtg: extra package path: %s\n", options.ErrorArg)
	case driver.ParseErrWindowsGUIRequiresWindows:
		fmt.Fprintln(os.Stderr, "rtg: -windows-gui requires a Windows target")
	case driver.ParseErrMixedFileList:
		fmt.Fprintf(os.Stderr, "rtg: explicit source list contains a non-.go argument: %s\n", options.ErrorArg)
	default:
		fmt.Fprintf(os.Stderr, "rtg: option parse failed with error %d\n", options.Error)
	}
}
