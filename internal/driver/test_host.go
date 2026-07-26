//go:build !renvo

package driver

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"renvo.dev/internal/load"
	"renvo.dev/internal/testfront"
)

const (
	TestOK = iota
	TestErrArguments
	TestErrWorkDir
	TestErrGenerate
	TestErrBackend
	TestErrCompile
	TestErrExecute
)

const TestHelpText = "Usage: renvo test [-s] [-tags <list>] [-arena-size <bytes>] [package]\nCompile and run ordinary TestXxx functions from package _test.go files using Renvo.\n"

type TestResult struct {
	Compile  CompileResult
	Ok       bool
	Error    int
	ErrorArg string
	ExitCode int
}

func TestCommandRequested(args []string) bool {
	return len(args) > 1 && args[1] == "test"
}

func RunTestCommand(args []string, env []string, backend Backend, stdin io.Reader, stdout, stderr io.Writer) TestResult {
	result := TestResult{Ok: true}
	compileArgs, packageDir, parseErr, errorArg := parseTestCommand(args)
	if parseErr != TestOK {
		return testFail(result, parseErr, errorArg)
	}
	target := hostTarget()
	if target == "" {
		return testFail(result, TestErrExecute, runtime.GOOS+"/"+runtime.GOARCH)
	}
	absoluteDir, err := filepath.Abs(packageDir)
	if err != nil {
		return testFail(result, TestErrWorkDir, packageDir)
	}
	generated, err := testfront.GenerateRenvoPackage(absoluteDir)
	if err != nil {
		return testFail(result, TestErrGenerate, err.Error())
	}
	files, cleanup, err := testfront.WriteTemporaryPackage(absoluteDir, generated)
	if err != nil {
		return testFail(result, TestErrGenerate, err.Error())
	}
	defer cleanup()
	for i := 0; i < len(files); i++ {
		compileArgs = append(compileArgs, files[i])
	}
	compileArgs = append(compileArgs, "-t", target, "-o", "-")
	if backend == nil {
		commandBackend, ok := CommandBackendFromEnv(env)
		if !ok {
			return testFail(result, TestErrBackend, "")
		}
		backend = commandBackend
	}
	compiled := CompileFromFSWithModuleCache(
		compileArgs, load.CleanPath(absoluteDir), StdRootFromEnv(env),
		ModuleCacheFromEnv(env), OSFS{}, backend,
	)
	result.Compile = compiled
	if !compiled.Ok {
		return testFail(result, TestErrCompile, "")
	}
	runDir, err := os.MkdirTemp("", "renvo-test-run-*")
	if err != nil {
		return testFail(result, TestErrExecute, err.Error())
	}
	defer os.RemoveAll(runDir)
	name := "test"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	executable := filepath.Join(runDir, name)
	if err := os.WriteFile(executable, compiled.Binary, 0o755); err != nil {
		return testFail(result, TestErrExecute, err.Error())
	}
	command := exec.Command(executable)
	command.Env = env
	command.Stdin = stdin
	command.Stdout = stdout
	command.Stderr = stderr
	err = command.Run()
	if err == nil {
		return result
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		return result
	}
	return testFail(result, TestErrExecute, err.Error())
}

func parseTestCommand(args []string) (compileArgs []string, packageDir string, err int, errorArg string) {
	if len(args) < 2 {
		return nil, "", TestErrArguments, "missing test command"
	}
	packageDir = "."
	packageSet := false
	for i := 2; i < len(args); i++ {
		arg := args[i]
		if arg == "--help" || arg == "-h" {
			return nil, "", TestErrArguments, "help"
		}
		if arg == "-s" {
			compileArgs = append(compileArgs, arg)
			continue
		}
		if arg == "-tags" || arg == "-arena-size" {
			if i+1 >= len(args) {
				return nil, "", TestErrArguments, arg
			}
			compileArgs = append(compileArgs, arg, args[i+1])
			i++
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			return nil, "", TestErrArguments, arg
		}
		if packageSet {
			return nil, "", TestErrArguments, arg
		}
		packageDir = arg
		packageSet = true
	}
	return compileArgs, packageDir, TestOK, ""
}

func testFail(result TestResult, err int, arg string) TestResult {
	result.Ok = false
	result.Error = err
	result.ErrorArg = arg
	if result.ExitCode == 0 {
		result.ExitCode = 1
	}
	return result
}
