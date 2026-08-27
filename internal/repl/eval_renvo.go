//go:build renvo

package repl

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/backendbridge"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/linkedimage"
	"renvo.dev/internal/load"
)

const replDiagnosticCapacity = 8192
const replImageArenaSize = 4 * 1024 * 1024

var replDiagnosticBuffer []byte

type EvalResult struct {
	Compiled   bool
	ExitCode   int
	Diagnostic []byte
}

// Session owns the in-process linker state for one REPL. Compiler scratch is
// reclaimed after every submission, while executable mappings and exported
// variable slots remain live until Reset.
type Session struct {
	linker driver.LinkedImageSession
}

func (s *Session) Reset() {
	s.linker.Reset()
}

type sourceFS struct {
	base         driver.SourceFS
	sourcePath   string
	source       []byte
	modulePath   string
	moduleSource []byte
}

func (fs sourceFS) ReadDir(path string) ([]driver.DirEntry, bool) {
	entries, ok := fs.base.ReadDir(path)
	return entries, ok
}

func (fs sourceFS) ReadFile(path string) ([]byte, bool) {
	path = load.CleanPath(path)
	if path == fs.sourcePath {
		return fs.source, true
	}
	if fs.modulePath != "" && path == fs.modulePath {
		return fs.moduleSource, true
	}
	data, ok := fs.base.ReadFile(path)
	return data, ok
}

func (fs sourceFS) PathExists(path string) bool {
	path = load.CleanPath(path)
	if path == fs.sourcePath || fs.modulePath != "" && path == fs.modulePath {
		return true
	}
	return fs.base.PathExists(path)
}

// Evaluate compiles one linked-image generation, migrates matching live
// symbols, and enters only that generation's main function.
func (s *Session) Evaluate(source []byte, env []string) EvalResult {
	if len(replDiagnosticBuffer) == 0 {
		replDiagnosticBuffer = make([]byte, replDiagnosticCapacity)
	}
	target := replTarget()
	targetID := replTargetID()
	if target == "" || targetID == 0 {
		return replDiagnostic("renvorepl: this host target cannot execute linked images\n")
	}
	workDir := replEnvValue(env, "PWD")
	if workDir == "" {
		workDir = "."
	}
	workDir = load.CleanPath(workDir)
	base := driver.RenvoFS{}
	fs := sourceFS{
		base:       base,
		sourcePath: load.JoinPath(workDir, "renvo_repl_input.go"),
		source:     source,
	}
	if !replHasModule(workDir, base) {
		fs.modulePath = load.JoinPath(workDir, "go.mod")
		fs.moduleSource = []byte("module renvo.dev/repl\n")
	}
	stdRoot := replEnvValue(env, "RENVO_STDROOT")
	if stdRoot == "" {
		stdRoot = "/std"
	}
	moduleCache := replEnvValue(env, "RENVO_MODCACHE")
	if moduleCache == "" {
		moduleCache = "/modules"
	}

	s.linker.Prepare()
	mark := arena.Mark()
	built := driver.BuildFromFSWithModuleCache(
		[]string{"-t", target, "-s", "-emit-image", "-o", "-", "renvo_repl_input.go"},
		workDir, stdRoot, moduleCache, fs,
	)
	if !built.Ok {
		return replBuildFailure(built.Diagnostic, mark)
	}

	persistMark := arena.PersistMark()
	unit := arena.PersistBytes(built.Unit)
	backendMark := mark
	remainder := backendMark % 4096
	if remainder != 0 {
		backendMark += 4096 - remainder
	}
	arena.Reset(backendMark)
	image, compiled := backendbridge.CompileUnitToImage(unit, target, true, replImageArenaSize, "")
	if !compiled {
		arena.Reset(mark)
		arena.PersistReset(persistMark)
		return replDiagnostic("renvorepl: backend compilation failed\n")
	}
	imageTarget, _, native, valid := linkedimage.Payload(image)
	if !valid || imageTarget != targetID || len(native) == 0 {
		arena.Reset(mark)
		arena.PersistReset(persistMark)
		return replDiagnostic("renvorepl: backend returned an invalid linked image\n")
	}
	exitCode := s.linker.Run(native, "renvorepl", nil, env)
	arena.Reset(mark)
	arena.PersistReset(persistMark)
	if exitCode < 0 {
		return replDiagnostic("renvorepl: failed to execute linked image\n")
	}
	return EvalResult{Compiled: true, ExitCode: exitCode}
}

// Evaluate is the one-shot form used by small embedding callers. Interactive
// consumers should retain a Session.
func Evaluate(source []byte, env []string) EvalResult {
	var session Session
	result := session.Evaluate(source, env)
	session.Reset()
	return result
}

func replBuildFailure(diagnostic driver.Diagnostic, mark int) EvalResult {
	result := replDiagnostic(driver.FormatDiagnostic(diagnostic))
	arena.Reset(mark)
	return result
}

func replDiagnostic(text string) EvalResult {
	used := len(text)
	if used > replDiagnosticCapacity {
		used = replDiagnosticCapacity
	}
	for i := 0; i < used; i++ {
		replDiagnosticBuffer[i] = text[i]
	}
	if used == replDiagnosticCapacity && used > 0 {
		replDiagnosticBuffer[used-1] = '\n'
	}
	return EvalResult{Diagnostic: replDiagnosticBuffer[:used]}
}

func replHasModule(workDir string, fs driver.SourceFS) bool {
	dir := workDir
	for {
		if _, ok := fs.ReadFile(load.JoinPath(dir, "go.mod")); ok {
			return true
		}
		next := load.DirPath(dir)
		if next == dir || dir == "." || dir == "/" {
			return false
		}
		dir = next
	}
}
