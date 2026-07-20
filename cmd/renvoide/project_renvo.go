//go:build renvo

package main

import (
	"renvo.dev/internal/driver"
	renvoos "renvo.dev/std/os"
	"renvo.dev/std/process"
)

type ideBuildSession struct {
	compiler *driver.RenvoCommandSession
	output   string
	done     bool
	result   projectActionResult
}

func beginCompileIDEProject(root, output, target string, env []string) *ideBuildSession {
	session := &ideBuildSession{output: output}
	if target == "" {
		session.done = true
		session.result = projectActionResult{message: "Build failed: select a Renvo target.", ok: false}
		return session
	}
	args := []string{"renvo", "-t", target, "-s", "-o", output, "."}
	session.compiler = driver.BeginRenvoCommand(args, projectEnvironment(env, root))
	return session
}

func (s *ideBuildSession) Step() (bool, projectActionResult) {
	if s == nil {
		return true, projectActionResult{message: "Build failed: compiler session is unavailable.", ok: false}
	}
	if s.done {
		return true, s.result
	}
	if !s.compiler.Step() {
		return false, projectActionResult{}
	}
	status, diagnostic := s.compiler.Result()
	if status != 0 {
		if diagnostic == "" {
			diagnostic = "Build failed."
		}
		s.result = projectActionResult{message: diagnostic, ok: false}
		s.done = true
		return true, s.result
	}
	file, err := renvoos.Open(s.output)
	if err != nil {
		s.result = projectActionResult{message: "Build failed: the compiler did not create " + s.output + ".", ok: false}
	} else {
		file.Close()
		s.result = projectActionResult{message: "Build succeeded: " + s.output, ok: true}
	}
	s.done = true
	return true, s.result
}

func compileIDEProject(root, output, target string, env []string) projectActionResult {
	session := beginCompileIDEProject(root, output, target, env)
	for {
		done, result := session.Step()
		if done {
			return result
		}
	}
}

func defaultIDETarget() string { return currentIDETarget() }

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
