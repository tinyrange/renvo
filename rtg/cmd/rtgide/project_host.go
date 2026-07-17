//go:build !rtg

package main

import (
	"os"
	"os/exec"
	"runtime"

	"j5.nz/rtg/rtg/internal/driver"
)

func compileIDEProject(root, output string, env []string) projectActionResult {
	backend, ok := driver.CommandBackendFromEnv(env)
	if !ok {
		return projectActionResult{message: "Build failed: set RTG_BACKEND to a compiler backend.", ok: false}
	}
	target := hostIDETarget()
	if target == "" {
		return projectActionResult{message: "Build failed: this host is not an RTG target yet.", ok: false}
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
