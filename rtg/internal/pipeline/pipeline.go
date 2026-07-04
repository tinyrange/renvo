package pipeline

import (
	"j5.nz/rtg/rtg/internal/build"
	"j5.nz/rtg/rtg/internal/link"
	"j5.nz/rtg/rtg/internal/load"
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
}

func BuildUnit(workDir string, stdRoot string, arg string, files []load.SourceFile) Result {
	result := Result{
		Ok:           true,
		Error:        PipelineOK,
		ErrorPackage: -1,
		ErrorFile:    -1,
		ErrorToken:   -1,
	}
	workspace := load.LoadWorkspace(workDir, stdRoot, arg, files)
	result.Workspace = workspace
	if !workspace.Ok {
		return pipelineFail(result, PipelineErrLoad, workspace.ErrorFile, -1, -1)
	}
	built := build.BuildUnits(workspace.Graph)
	result.Build = built
	if !built.Ok {
		return pipelineFail(result, PipelineErrBuild, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	linked := link.LinkBuild(built)
	result.Link = linked
	if !linked.Ok {
		return pipelineFail(result, PipelineErrLink, linked.ErrorPackage, -1, -1)
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
