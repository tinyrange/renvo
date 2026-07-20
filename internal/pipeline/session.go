package pipeline

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/build"
	"renvo.dev/internal/link"
	"renvo.dev/internal/load"
)

// Session advances the frontend by one phase or package at a time. It keeps
// arena-owned state alive between calls so GUI applications can return to
// their event loop after every Step.
type Session struct {
	workDir    string
	stdRoot    string
	arg        string
	files      []load.SourceFile
	filesStart int
	filesEnd   int
	transient  bool
	cached     bool
	stage      int
	loadStart  int
	loadEnd    int
	builder    *build.ProgramSession
	linker     *link.PackageSession
	result     Result
}

func BeginSession(workDir string, stdRoot string, arg string, files []load.SourceFile, filesStart int, filesEnd int, transient bool, cached bool) *Session {
	return &Session{
		workDir:    workDir,
		stdRoot:    stdRoot,
		arg:        arg,
		files:      files,
		filesStart: filesStart,
		filesEnd:   filesEnd,
		transient:  transient,
		cached:     cached,
		result: Result{
			Ok:           true,
			Error:        PipelineOK,
			ErrorPackage: -1,
			ErrorFile:    -1,
			ErrorToken:   -1,
		},
	}
}

// Step returns true after success or failure. Each non-final call performs no
// more than workspace loading, header checking, one package build, linker
// preparation, or one package-artifact resolution.
func (s *Session) Step() bool {
	if s == nil || s.stage >= 4 {
		return true
	}
	if s.stage == 0 {
		s.loadStart = arena.Mark()
		workspace := load.LoadWorkspace(s.workDir, s.stdRoot, s.arg, s.files)
		s.loadEnd = arena.Mark()
		s.result.Workspace = workspace
		if !workspace.Ok {
			s.result = pipelineFail(s.result, PipelineErrLoad, -1, workspace.ErrorFile, -1)
			s.stage = 4
			return true
		}
		s.builder = build.BeginProgramsSession(workspace.Graph, s.transient, s.cached)
		s.stage = 1
		return false
	}
	if s.stage == 1 {
		if !s.builder.Step() {
			return false
		}
		built := s.builder.Result()
		s.result.Build = built
		if !built.Ok {
			s.result = pipelineFail(s.result, PipelineErrBuild, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
			s.stage = 4
			return true
		}
		s.linker = link.BeginPackageSession(built, s.transient)
		s.stage = 2
		return false
	}
	if s.stage == 2 {
		if !s.linker.Step() {
			return false
		}
		linked := s.linker.Result()
		s.result.Link = linked
		if !linked.Ok {
			s.result = pipelineFail(s.result, PipelineErrLink, linked.ErrorPackage, -1, -1)
			s.stage = 4
			return true
		}
		s.stage = 3
		return false
	}
	if s.transient {
		s.result.Workspace = load.Workspace{}
		s.result.Build = build.Result{}
		arena.Discard(s.loadStart, s.loadEnd)
		arena.Discard(s.filesStart, s.filesEnd)
	}
	s.stage = 4
	return true
}

func (s *Session) Result() Result {
	if s == nil {
		return Result{Ok: false, Error: PipelineErrLoad, ErrorPackage: -1, ErrorFile: -1, ErrorToken: -1}
	}
	return s.result
}
