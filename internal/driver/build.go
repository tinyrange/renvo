package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/load"
	"renvo.dev/internal/pipeline"
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
	Diagnostic   Diagnostic
}

func BuildUnit(args []string, workDir string, stdRoot string, files []load.SourceFile) BuildResult {
	result := newBuildResult()
	options := ParseOptions(args)
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
	}
	filtered, errorPath, sourceError := filterSourcesForOptions(files, workDir, options)
	if sourceError != SourceOK {
		result.Sources = SourceResult{Error: sourceError, ErrorPath: errorPath}
		return buildFail(result, BuildErrSource, "", errorPath, -1, -1, -1, -1)
	}
	rootArg := options.Package
	if len(options.Files) > 0 {
		rootArg = load.DirPath(load.JoinPath(workDir, options.Files[0]))
	}
	built := pipeline.BuildUnit(workDir, stdRoot, rootArg, filtered)
	result.Pipeline = built
	if !built.Ok {
		return buildFail(result, BuildErrPipeline, "", "", -1, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	result.Unit = built.Link.Data
	return result
}

func BuildFromFS(args []string, workDir string, stdRoot string, fs SourceFS) BuildResult {
	return buildFromFS(args, workDir, stdRoot, "", fs, false)
}

func buildFromFSCompact(args []string, workDir string, stdRoot string, fs SourceFS) BuildResult {
	return buildFromFS(args, workDir, stdRoot, "", fs, true)
}

func BuildFromFSWithModuleCache(args []string, workDir string, stdRoot string, moduleCache string, fs SourceFS) BuildResult {
	return buildFromFS(args, workDir, stdRoot, moduleCache, fs, false)
}

func buildFromFSCompactWithModuleCache(args []string, workDir string, stdRoot string, moduleCache string, fs SourceFS) BuildResult {
	return buildFromFS(args, workDir, stdRoot, moduleCache, fs, true)
}

func buildFromFS(args []string, workDir string, stdRoot string, moduleCache string, fs SourceFS, compact bool) BuildResult {
	result := newBuildResult()
	options := ParseOptions(args)
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
	}
	sourcesStart := arena.Mark()
	var sources SourceResult
	if len(options.Files) > 0 {
		sources = CollectSourceFilesForTargetTagsWithModuleCache(workDir, stdRoot, options.Files, options.Target, options.Tags, moduleCache, fs)
	} else {
		sources = CollectSourcesForTargetTagsWithModuleCache(workDir, stdRoot, options.Package, options.Target, options.Tags, moduleCache, fs)
	}
	sourcesEnd := arena.Mark()
	result.Sources = sources
	if !sources.Ok {
		return buildFail(result, BuildErrSource, "", sources.ErrorPath, -1, -1, -1, -1)
	}
	rootArg := options.Package
	if len(options.Files) > 0 {
		rootArg = sources.Root.Dir
	}
	var built pipeline.Result
	if compact {
		built = pipeline.BuildUnitWithTransientFiles(workDir, stdRoot, rootArg, sources.Files, sourcesStart, sourcesEnd)
	} else {
		built = pipeline.BuildUnit(workDir, stdRoot, rootArg, sources.Files)
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
	result.Diagnostic = diagnosticForBuild(result)
	return result
}
