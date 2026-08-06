//go:build !renvo

package difftest

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"renvo.dev/internal/backendcompiled"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
)

type Runner struct {
	StdRoot string
	Target  string
	Timeout time.Duration
}

type Execution struct {
	Compiled   bool
	Ran        bool
	TimedOut   bool
	Output     []byte
	ExitCode   int
	Diagnostic string
}

type Comparison struct {
	Host        Execution
	Renvo       Execution
	Interesting bool
	Signature   string
}

func HostTarget() string {
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
	default:
		return ""
	}
}

func (r Runner) Compare(source []byte) (comparison Comparison, err error) {
	if r.Target == "" {
		r.Target = HostTarget()
	}
	if r.Timeout <= 0 {
		r.Timeout = 3 * time.Second
	}
	if r.Target == "" {
		return comparison, fmt.Errorf("unsupported host %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if err := validateHostSource(source); err != nil {
		return comparison, fmt.Errorf("generated source rejected by host Go type checker: %w", err)
	}
	dir, err := os.MkdirTemp("", "renvo-difftest-*")
	if err != nil {
		return comparison, err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module difftest.generated\n\ngo 1.25\n"), 0o644); err != nil {
		return comparison, err
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), source, 0o644); err != nil {
		return comparison, err
	}

	comparison.Host = r.compileHost(dir)
	if !comparison.Host.Compiled {
		return comparison, fmt.Errorf("generated source rejected by host Go compiler: %s", comparison.Host.Diagnostic)
	}
	if !comparison.Host.Ran {
		return comparison, fmt.Errorf("generated source did not run successfully with host Go: %s", comparison.Host.Diagnostic)
	}
	comparison.Renvo = r.compileRenvo(dir)
	comparison.Interesting, comparison.Signature = compareExecutions(comparison.Host, comparison.Renvo)
	return comparison, nil
}

func validateHostSource(source []byte) error {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "main.go", source, parser.SkipObjectResolution)
	if err != nil {
		return err
	}
	configuration := types.Config{Importer: importer.Default(), Sizes: types.SizesFor("gc", runtime.GOARCH)}
	_, err = configuration.Check("difftest.generated", fileSet, []*ast.File{file}, nil)
	return err
}

func (r Runner) compileHost(dir string) Execution {
	binary := filepath.Join(dir, "host-program")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	contextValue, cancel := context.WithTimeout(context.Background(), maxDuration(10*time.Second, r.Timeout))
	defer cancel()
	command := exec.CommandContext(contextValue, "go", "build", "-trimpath", "-o", binary, ".")
	command.Dir = dir
	command.Env = appendFilteredEnv(os.Environ(), "GOWORK=off", "CGO_ENABLED=0")
	output, err := command.CombinedOutput()
	if err != nil {
		return Execution{TimedOut: contextValue.Err() == context.DeadlineExceeded, Diagnostic: strings.TrimSpace(string(output))}
	}
	return runProgram(binary, dir, r.Timeout)
}

func (r Runner) compileRenvo(dir string) (execution Execution) {
	defer func() {
		if recovered := recover(); recovered != nil {
			execution.Diagnostic = fmt.Sprintf("compiler panic: %v", recovered)
		}
	}()
	compiled := driver.CompileFromFS(
		[]string{"-t", r.Target, "-s", "-o", "app", "."},
		load.CleanPath(dir), load.CleanPath(r.StdRoot), driver.OSFS{}, backendcompiled.Backend{},
	)
	if !compiled.Ok {
		diagnostic := compiled.Diagnostic.Code
		if compiled.Diagnostic.Message != "" {
			diagnostic += ": " + compiled.Diagnostic.Message
		}
		return Execution{Diagnostic: diagnostic}
	}
	binary := filepath.Join(dir, "renvo-program")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if err := os.WriteFile(binary, compiled.Binary, 0o755); err != nil {
		return Execution{Diagnostic: err.Error()}
	}
	return runProgram(binary, dir, r.Timeout)
}

func runProgram(binary string, dir string, timeout time.Duration) Execution {
	contextValue, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(contextValue, binary)
	command.Dir = dir
	command.Env = []string{"PWD=" + dir}
	output, err := command.CombinedOutput()
	execution := Execution{Compiled: true, Output: output, ExitCode: 0}
	if contextValue.Err() == context.DeadlineExceeded {
		execution.TimedOut = true
		execution.Diagnostic = "execution timeout"
		return execution
	}
	if err != nil {
		execution.Diagnostic = err.Error()
		if exitError, ok := err.(*exec.ExitError); ok {
			execution.ExitCode = exitError.ExitCode()
		} else {
			execution.ExitCode = -1
		}
		return execution
	}
	execution.Ran = true
	return execution
}

func compareExecutions(host, renvo Execution) (bool, string) {
	if !renvo.Compiled {
		code := renvo.Diagnostic
		if index := strings.IndexByte(code, ':'); index >= 0 {
			code = code[:index]
		}
		if code == "" {
			code = "unknown"
		}
		return true, "renvo-compile:" + code
	}
	if renvo.TimedOut != host.TimedOut {
		return true, "execution-timeout"
	}
	if renvo.ExitCode != host.ExitCode {
		return true, "exit-code"
	}
	if !bytes.Equal(renvo.Output, host.Output) {
		return true, "output"
	}
	return false, ""
}

func appendFilteredEnv(environment []string, values ...string) []string {
	for _, value := range values {
		key := value[:strings.IndexByte(value, '=')+1]
		filtered := environment[:0]
		for _, item := range environment {
			if !strings.HasPrefix(item, key) {
				filtered = append(filtered, item)
			}
		}
		environment = append(filtered, value)
	}
	return environment
}

func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
