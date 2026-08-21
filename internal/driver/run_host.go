//go:build !renvo

package driver

import (
	"io"
	"os"
	"runtime"

	"renvo.dev/internal/linkedimage"
	"renvo.dev/internal/load"
	"renvo.dev/internal/runimage"
)

const (
	RunOK = iota
	RunErrArguments
	RunErrWorkDir
	RunErrBackend
	RunErrCompile
	RunErrImage
	RunErrExecute
)

type RunResult struct {
	Compile  CompileResult
	Ok       bool
	Error    int
	ErrorArg string
	ExitCode int
	Loader   string
}

func ScriptCommandRequested(args []string) bool {
	return len(args) > 1 && args[1] == "run"
}

func RunScriptCommand(args []string, env []string, backend Backend, stdin io.Reader, stdout, stderr io.Writer) RunResult {
	result := RunResult{Ok: true}
	compileArgs, programArgs, argError, errorArg := parseRunCommand(args)
	if argError != RunOK {
		return runFail(result, argError, errorArg)
	}
	target := hostTarget()
	if target == "" {
		return runFail(result, RunErrExecute, runtime.GOOS+"/"+runtime.GOARCH)
	}
	script := compileArgs[len(compileArgs)-1]
	compileArgs = append(compileArgs, "-script")
	compileArgs = append(compileArgs, "-emit-image")
	compileArgs = append(compileArgs, "-t", target, "-o", "-")
	if backend == nil {
		return runFail(result, RunErrBackend, "")
	}
	workDir, err := os.Getwd()
	if err != nil {
		return runFail(result, RunErrWorkDir, "")
	}
	compiled := CompileFromFSWithModuleCache(
		compileArgs, load.CleanPath(workDir), StdRootFromEnv(env),
		ModuleCacheFromEnv(env), OSFS{}, backend,
	)
	result.Compile = compiled
	if !compiled.Ok {
		return runFail(result, RunErrCompile, "")
	}
	image, err := linkedimage.Decode(compiled.Binary)
	if err != nil || image.Target != target {
		return runFail(result, RunErrImage, target)
	}
	executed := runimage.Run(image, script, programArgs, env, stdin, stdout, stderr)
	result.ExitCode = executed.ExitCode
	result.Loader = executed.Loader
	if executed.Err != nil {
		return runFail(result, RunErrExecute, executed.Err.Error())
	}
	return result
}

func parseRunCommand(args []string) (compileArgs []string, programArgs []string, err int, errorArg string) {
	if len(args) < 3 {
		return nil, nil, RunErrArguments, "missing script"
	}
	i := 2
	script := ""
	for i < len(args) {
		arg := args[i]
		if script != "" {
			if arg == "--" {
				programArgs = append(programArgs, args[i+1:]...)
			} else {
				programArgs = append(programArgs, args[i:]...)
			}
			break
		}
		if arg == "--help" || arg == "-h" {
			return nil, nil, RunErrArguments, "help"
		}
		if arg == "-s" {
			compileArgs = append(compileArgs, arg)
			i++
			continue
		}
		if arg == "-tags" || arg == "-arena-size" {
			if i+1 >= len(args) {
				return nil, nil, RunErrArguments, arg
			}
			compileArgs = append(compileArgs, arg, args[i+1])
			i += 2
			continue
		}
		if len(arg) > 0 && arg[0] == '-' {
			return nil, nil, RunErrArguments, arg
		}
		if !optionArgIsGoFile(arg) {
			return nil, nil, RunErrArguments, arg
		}
		script = arg
		compileArgs = append(compileArgs, script)
		i++
	}
	if script == "" {
		return nil, nil, RunErrArguments, "missing script"
	}
	return compileArgs, programArgs, RunOK, ""
}

func hostTarget() string {
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return "linux/amd64"
	case "linux/386":
		return "linux/386"
	case "linux/arm64":
		return "linux/aarch64"
	case "linux/arm":
		return "linux/arm"
	case "windows/amd64":
		return "windows/amd64"
	case "windows/386":
		return "windows/386"
	case "windows/arm64":
		return "windows/arm64"
	case "darwin/arm64":
		return "darwin/arm64"
	case "freebsd/amd64":
		return "freebsd/amd64"
	case "openbsd/amd64":
		return "openbsd/amd64"
	case "netbsd/amd64":
		return "netbsd/amd64"
	default:
		return ""
	}
}

func runFail(result RunResult, err int, arg string) RunResult {
	result.Ok = false
	result.Error = err
	result.ErrorArg = arg
	if result.ExitCode == 0 {
		result.ExitCode = 1
	}
	return result
}
