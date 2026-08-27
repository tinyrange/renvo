//go:build !renvo

package bootstrap

import (
	"fmt"
	"io"
	"os"

	"renvo.dev/internal/driver"
)

func Run(args []string, env []string, backend driver.Backend) int {
	args, backend = bootstrapArgs(args, backend)
	if response := driver.ExpandCCompilerResponseFiles(args, driver.OSFS{}); response.Ok {
		args = response.Args
	} else {
		fmt.Fprintf(os.Stderr, "renvo cc: could not read response file: %s\n", response.ErrorPath)
		return 1
	}
	if request := driver.InspectCCompilerRequest(args); request.Kind != driver.CCompilerRequestNone {
		var input []byte
		if request.Kind == driver.CCompilerRequestPreprocessStdin {
			var err error
			input, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, "renvo cc: failed to read standard input")
				return 1
			}
		}
		status, output := driver.ExecuteCCompilerRequest(request, input)
		if status == 0 {
			fmt.Fprint(os.Stdout, output)
		} else {
			fmt.Fprint(os.Stderr, output)
		}
		return status
	}
	args = driver.NormalizeCCompilerCommand(args)
	if driver.CAssemblyCommandRequested(args) {
		workDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "renvo cc: failed to determine working directory")
			return 1
		}
		result := driver.CompileCAssemblyCommand(args, workDir, driver.OSFS{})
		if !result.Ok {
			fmt.Fprint(os.Stderr, driver.FormatDiagnostic(driver.CAssemblyCommandDiagnostic(result)))
			return 1
		}
		if result.Output == "-" {
			_, err = os.Stdout.Write(result.Source)
		} else {
			err = os.WriteFile(result.Output, result.Source, 0o644)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "renvo cc: failed to write assembly output")
			return 1
		}
		if len(result.DependencyData) > 0 {
			if err := os.WriteFile(result.DependencyFile, result.DependencyData, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "renvo cc: failed to write dependency file")
				return 1
			}
		}
		return 0
	}
	if driver.CPreprocessCommandRequested(args) {
		workDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "renvo cc: failed to determine working directory")
			return 1
		}
		var input []byte
		if driver.CPreprocessCommandUsesStandardInput(args) {
			input, err = io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, "renvo cc: failed to read standard input")
				return 1
			}
		}
		result := driver.PreprocessCCommandWithInput(args, workDir, driver.OSFS{}, input)
		if !result.Ok {
			fmt.Fprint(os.Stderr, driver.FormatDiagnostic(driver.CPreprocessCommandDiagnostic(result)))
			return 1
		}
		if result.Output == "-" {
			_, err = os.Stdout.Write(result.Source)
		} else {
			err = os.WriteFile(result.Output, result.Source, 0o644)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "renvo cc: failed to write preprocessor output")
			return 1
		}
		if len(result.DependencyData) > 0 {
			if err := os.WriteFile(result.DependencyFile, result.DependencyData, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "renvo cc: failed to write dependency file")
				return 1
			}
		}
		return 0
	}
	if driver.CSyntaxCommandRequested(args) {
		workDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, "renvo cc: failed to determine working directory")
			return 1
		}
		result := driver.CheckCCommand(args, workDir, driver.OSFS{})
		if !result.Ok {
			fmt.Fprint(os.Stderr, driver.FormatDiagnostic(driver.CSyntaxCommandDiagnostic(result)))
			return 1
		}
		if len(result.DependencyData) > 0 {
			if err := os.WriteFile(result.DependencyFile, result.DependencyData, 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "renvo cc: failed to write dependency file")
				return 1
			}
		}
		return 0
	}
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
	if result.Error == driver.CompileErrBuild {
		printBuildError(result.Build)
		return
	}
	fmt.Fprint(os.Stderr, driver.FormatDiagnostic(result.Diagnostic))
}

func printBuildError(result driver.BuildResult) {
	fmt.Fprint(os.Stderr, driver.FormatDiagnostic(result.Diagnostic))
}
