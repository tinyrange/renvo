//go:build rtg

package main

import (
	"j5.nz/rtg/rtg/internal/driver"
	rtgos "j5.nz/rtg/rtg/std/os"
	"j5.nz/rtg/rtg/std/process"
)

func compileIDEProject(root, output string, env []string) projectActionResult {
	target := currentIDETarget()
	if target == "" {
		return projectActionResult{message: "Build failed: this host is not an RTG target yet.", ok: false}
	}
	args := []string{"rtg", "-t", target, "-s", "-o", output, "."}
	buildEnv := projectEnvironment(env, root)
	if driver.RunRTGCommand(args, buildEnv) != 0 {
		return projectActionResult{message: "Build failed. See the compiler diagnostic on stderr.", ok: false}
	}
	file, err := rtgos.Open(output)
	if err != nil {
		return projectActionResult{message: "Build failed: the compiler did not create " + output + ".", ok: false}
	}
	file.Close()
	return projectActionResult{message: "Build succeeded: " + output, ok: true}
}

func launchIDEProject(output, root string) projectActionResult {
	if !process.Start(output, root) {
		return projectActionResult{message: "Run failed: could not start " + output, ok: false}
	}
	return projectActionResult{message: "Application launched: " + output, ok: true}
}

func projectEnvironment(env []string, root string) []string {
	out := make([]string, 0, len(env)+1)
	for i := 0; i < len(env); i++ {
		if !workspaceHasPrefix(env[i], "PWD=") {
			out = append(out, env[i])
		}
	}
	out = append(out, "PWD="+root)
	return out
}

func workspaceHasPrefix(value, prefix string) bool {
	if len(value) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if value[i] != prefix[i] {
			return false
		}
	}
	return true
}
