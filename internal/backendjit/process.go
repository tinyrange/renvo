//go:build !renvo

package backendjit

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"renvo.dev/internal/driver"
	"renvo.dev/internal/rtgb"
	"renvo.dev/internal/unit"
)

func runNativeCompilerPayload(artifact rtgb.Artifact, request Request) driver.BackendResult {
	directory, err := os.MkdirTemp("", "renvo-derived-compiler-*")
	if err != nil {
		return backendIOError("create cached compiler directory")
	}
	defer func() { _ = os.RemoveAll(directory) }()
	name := "compiler"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(directory, name)
	if err = os.WriteFile(path, artifact.Payload, 0o700); err != nil {
		return backendIOError("materialize cached compiler")
	}
	if err = os.Chmod(path, 0o700); err != nil && runtime.GOOS != "windows" {
		return backendIOError("permission cached compiler")
	}
	return runNativeCompiler(path, artifact, request)
}

// runNativeCompiler executes one cached target compiler as an ordinary host
// process. The process boundary is deliberately one request per invocation:
// exactly one RenvoUnit enters on stdin and only the compiled artifact leaves
// on stdout.
func runNativeCompiler(path string, artifact rtgb.Artifact, request Request) driver.BackendResult {
	if !compatibleRequest(artifact, request) {
		return incompatibleProtocol()
	}
	if path == "" {
		return backendIOError("start cached compiler")
	}
	command := exec.Command(path, compilerRequestArgs(artifact.Descriptor.Name, request.Options, "-", "-")...)
	command.Stdin = bytes.NewReader(request.Unit)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if err != nil {
		message := "cached compiler execution failed: " + err.Error()
		if detail := strings.TrimSpace(stderr.String()); detail != "" {
			message += ": " + detail
		}
		code := "RENVO-BACKEND-009"
		if _, started := err.(*exec.ExitError); !started {
			code = "RENVO-BACKEND-011"
		}
		return driver.BackendResult{Diagnostic: driver.Diagnostic{
			Phase: "backend", Code: code, Message: message,
		}}
	}
	if stdout.Len() == 0 {
		return backendIOError("read cached compiler output")
	}
	return driver.BackendResult{Binary: stdout.Bytes(), Ok: true}
}

func compatibleRequest(artifact rtgb.Artifact, request Request) bool {
	return request.Protocol == ProtocolVersion && artifact.Protocol == ProtocolVersion &&
		artifact.Unit == unit.Version && artifact.Optimization == OptimizationVersion
}

func incompatibleProtocol() driver.BackendResult {
	return driver.BackendResult{Diagnostic: driver.Diagnostic{
		Phase: "backend", Code: "RENVO-BACKEND-008", Message: "prepared backend protocol is incompatible",
	}}
}

func compilerRequestArgs(target string, options driver.BackendCompileOptions, output string, input string) []string {
	args := []string{"-t", target}
	if options.Strip {
		args = append(args, "-s")
	}
	if options.WindowsGUI {
		args = append(args, "-windows-gui")
	}
	if options.EmitImage {
		args = append(args, "-emit-image")
	}
	if options.ObjectFile || options.Mode == driver.ModeObject {
		args = append(args, "-object")
	}
	if options.ArenaSize > 0 {
		args = append(args, "-arena-size", decimal(options.ArenaSize))
	}
	if options.Output != "" {
		args = append(args, "-module-name", options.Output)
	}
	if options.ModuleLicense != "" {
		args = append(args, "-module-license", options.ModuleLicense)
	}
	return append(args, "-o", output, input)
}
