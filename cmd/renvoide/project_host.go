//go:build !renvo

package main

import (
	"os"
	"os/exec"
	"runtime"

	"renvo.dev/internal/driver"
)

type ideBuildSession struct {
	root   string
	output string
	target string
	env    []string
	done   bool
	result projectActionResult
}

func beginCompileIDEProject(root, output, target string, env []string) *ideBuildSession {
	return &ideBuildSession{root: root, output: output, target: target, env: env}
}

func (s *ideBuildSession) Step() (bool, projectActionResult) {
	if s == nil {
		return true, projectActionResult{message: "Build failed: compiler session is unavailable.", ok: false}
	}
	if s.done {
		return true, s.result
	}
	s.result = compileIDEProjectNow(s.root, s.output, s.target, s.env)
	s.done = true
	return true, s.result
}

func compileIDEProject(root, output, target string, env []string) projectActionResult {
	session := beginCompileIDEProject(root, output, target, env)
	_, result := session.Step()
	return result
}

func compileIDEProjectNow(root, output, target string, env []string) projectActionResult {
	backend, ok := driver.CommandBackendFromEnv(env)
	if !ok {
		return projectActionResult{message: "Build failed: set RENVO_BACKEND to a compiler backend.", ok: false}
	}
	if target == "" {
		return projectActionResult{message: "Build failed: select a Renvo target.", ok: false}
	}
	args := []string{"-t", target, "-s", "-o", output, "."}
	compiled := driver.CompileFromFSWithModuleCache(args, root, driver.StdRootFromEnv(env), driver.EnvValue(env, driver.ModuleCacheEnv), driver.OSFS{}, backend)
	if !compiled.Ok {
		message := "Build failed."
		if compiled.Diagnostic.Valid() {
			message = driver.FormatDiagnostic(compiled.Diagnostic)
		}
		return projectActionResult{message: message, ok: false}
	}
	if err := os.WriteFile(output, compiled.Binary, 0755); err != nil {
		return projectActionResult{message: "Build failed while writing " + output + ".", ok: false}
	}
	return projectActionResult{message: "Build succeeded: " + output, ok: true}
}

func defaultIDETarget() string { return hostIDETarget() }

func launchIDEProject(output, root string) projectActionResult {
	command := exec.Command(output)
	command.Dir = root
	if err := command.Start(); err != nil {
		return projectActionResult{message: "Run failed: " + err.Error(), ok: false}
	}
	if command.Process != nil {
		_ = command.Process.Release()
	}
	return projectActionResult{message: "Application launched: " + output, ok: true}
}

func hostIDETarget() string {
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		return "darwin/arm64"
	}
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		return "linux/amd64"
	}
	if runtime.GOOS == "linux" && runtime.GOARCH == "386" {
		return "linux/386"
	}
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		return "windows/amd64"
	}
	if runtime.GOOS == "windows" && runtime.GOARCH == "386" {
		return "windows/386"
	}
	if runtime.GOOS == "windows" && runtime.GOARCH == "arm64" {
		return "windows/arm64"
	}
	return ""
}
