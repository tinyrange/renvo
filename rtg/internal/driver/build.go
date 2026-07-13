package driver

import (
	"j5.nz/rtg/rtg/internal/arena"
	"j5.nz/rtg/rtg/internal/load"
	"j5.nz/rtg/rtg/internal/pipeline"
)

const (
	BuildOK = iota
	BuildErrOptions
	BuildErrSource
	BuildErrPipeline
)

type BuildResult struct {
	Options      Options
	Sources      SourceResult
	Pipeline     pipeline.Result
	Unit         []byte
	Ok           bool
	Error        int
	ErrorArg     string
	ErrorPath    string
	ErrorAt      int
	ErrorPackage int
	ErrorFile    int
	ErrorToken   int
}

func BuildUnit(args []string, workDir string, stdRoot string, files []load.SourceFile) BuildResult {
	result := newBuildResult()
	options := ParseOptions(args)
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
	}
	built := pipeline.BuildUnit(workDir, stdRoot, options.Package, filterSourcesForTarget(files, options.Target))
	result.Pipeline = built
	if !built.Ok {
		return buildFail(result, BuildErrPipeline, "", "", -1, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	result.Unit = built.Link.Data
	return result
}

func BuildFromFS(args []string, workDir string, stdRoot string, fs SourceFS) BuildResult {
	return buildFromFS(args, workDir, stdRoot, fs, false)
}

func buildFromFSCompact(args []string, workDir string, stdRoot string, fs SourceFS) BuildResult {
	return buildFromFS(args, workDir, stdRoot, fs, true)
}

func buildFromFS(args []string, workDir string, stdRoot string, fs SourceFS, compact bool) BuildResult {
	result := newBuildResult()
	options := ParseOptions(args)
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
	}
	sourcesStart := arena.Mark()
	sources := CollectSourcesForTarget(workDir, stdRoot, options.Package, options.Target, fs)
	sourcesEnd := arena.Mark()
	result.Sources = sources
	if !sources.Ok {
		return buildFail(result, BuildErrSource, "", sources.ErrorPath, -1, -1, -1, -1)
	}
	var built pipeline.Result
	if compact {
		built = pipeline.BuildUnitWithTransientFiles(workDir, stdRoot, options.Package, sources.Files, sourcesStart, sourcesEnd)
	} else {
		built = pipeline.BuildUnit(workDir, stdRoot, options.Package, sources.Files)
	}
	result.Pipeline = built
	if !built.Ok {
		return buildFail(result, BuildErrPipeline, "", "", -1, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	result.Unit = built.Link.Data
	if compact {
		result.Sources = SourceResult{}
	}
	return result
}

func newBuildResult() BuildResult {
	var result BuildResult
	result.Ok = true
	result.Error = BuildOK
	result.ErrorAt = -1
	result.ErrorPackage = -1
	result.ErrorFile = -1
	result.ErrorToken = -1
	return result
}

func buildFail(result BuildResult, err int, arg string, path string, at int, pkg int, file int, tok int) BuildResult {
	result.Ok = false
	result.Error = err
	result.ErrorArg = arg
	result.ErrorPath = path
	result.ErrorAt = at
	result.ErrorPackage = pkg
	result.ErrorFile = file
	result.ErrorToken = tok
	return result
}
