package pipeline

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/build"
	"renvo.dev/internal/link"
	"renvo.dev/internal/load"
)

const (
	PipelineOK = iota
	PipelineErrLoad
	PipelineErrBuild
	PipelineErrLink
)

type Result struct {
	Workspace    load.Workspace
	Build        build.Result
	Link         link.Result
	Ok           bool
	Error        int
	ErrorPackage int
	ErrorFile    int
	ErrorToken   int
	ErrorPath    string
	ErrorOffset  int
}

// Config supplies target semantics and storage ownership for one build.
type Config struct {
	Layout     load.TargetLayout
	Transient  bool
	Cached     bool
	Object     bool
	FilesStart int
	FilesEnd   int
}

func BuildUnitConfigured(workDir string, stdRoot string, arg string, files []load.SourceFile, config Config) Result {
	if !config.Cached {
		return buildUnitDirect(workDir, stdRoot, arg, files, config.FilesStart, config.FilesEnd, config.Transient, config.Object, config.Layout)
	}
	session := beginSession(workDir, stdRoot, arg, files, config.FilesStart, config.FilesEnd, config.Transient, true, config.Object)
	session.SetTargetLayout(config.Layout)
	for !session.Step() {
	}
	return session.Result()
}

func BuildUnit(workDir string, stdRoot string, arg string, files []load.SourceFile) Result {
	return BuildUnitConfigured(workDir, stdRoot, arg, files, Config{})
}

func BuildObjectUnit(workDir string, stdRoot string, arg string, files []load.SourceFile) Result {
	return BuildUnitConfigured(workDir, stdRoot, arg, files, Config{Object: true})
}

// BuildUnitWithTransientFiles allows the command driver to release source
// collection storage once lowering has copied every package into link units.
func BuildUnitWithTransientFiles(workDir string, stdRoot string, arg string, files []load.SourceFile, filesStart int, filesEnd int) Result {
	return BuildUnitConfigured(workDir, stdRoot, arg, files, Config{Transient: true, FilesStart: filesStart, FilesEnd: filesEnd})
}

// BuildUnitWithTransientFilesCached reuses unchanged lowered dependencies for
// repeated embedded builds while always rebuilding the root package.
func BuildUnitWithTransientFilesCached(workDir string, stdRoot string, arg string, files []load.SourceFile, filesStart int, filesEnd int) Result {
	session := BeginSession(workDir, stdRoot, arg, files, filesStart, filesEnd, true, true)
	for !session.Step() {
	}
	return session.Result()
}

func buildUnitDirect(workDir string, stdRoot string, arg string, files []load.SourceFile, filesStart int, filesEnd int, transient bool, object bool, layout load.TargetLayout) Result {
	result := Result{
		Ok:           true,
		Error:        PipelineOK,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	}
	loadStart := arena.Mark()
	workspace := load.LoadWorkspace(workDir, stdRoot, arg, files)
	workspace.Graph.Layout = layout
	loadEnd := arena.Mark()
	result.Workspace = workspace
	if !workspace.Ok {
		return pipelineFail(result, PipelineErrLoad, -1, workspace.ErrorFile, -1)
	}
	var built build.Result
	if object {
		built = build.BuildObjectPrograms(workspace.Graph)
	} else if transient {
		built = build.BuildProgramsTransient(workspace.Graph)
	} else {
		built = build.BuildPrograms(workspace.Graph)
	}
	result.Build = built
	if !built.Ok {
		result.ErrorPath = built.ErrorPath
		result.ErrorOffset = built.ErrorOffset
		return pipelineFail(result, PipelineErrBuild, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	var linked link.Result
	if object {
		linked = link.LinkBuildObjectCore(built)
	} else if transient {
		linked = link.LinkBuildCoreTransient(built)
	} else {
		linked = link.LinkBuildCore(built)
	}
	result.Link = linked
	if !linked.Ok {
		return pipelineFail(result, PipelineErrLink, linked.ErrorPackage, -1, -1)
	}
	if transient {
		// Successful transient builds expose only the linked result. Parsed and
		// lowered values would otherwise point into the released arena ranges.
		result.Workspace = load.Workspace{}
		result.Build = build.Result{}
		arena.Discard(loadStart, loadEnd)
		arena.Discard(filesStart, filesEnd)
	}
	return result
}

func pipelineFail(result Result, err int, pkg int, file int, tok int) Result {
	result.Ok = false
	result.Error = err
	result.ErrorPackage = pkg
	result.ErrorFile = file
	result.ErrorToken = tok
	return result
}
