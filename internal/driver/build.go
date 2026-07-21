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
	CacheKeyA    int
	CacheKeyB    int
	CacheHit     bool
}

var embeddedBuildCacheValid bool
var embeddedBuildCacheKeyA int
var embeddedBuildCacheKeyB int

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

// buildFromFSOneShotCompactWithModuleCache keeps transient arena reclamation
// without initializing persistent incremental-build state. A command-line
// compiler process performs one build, so populating editor caches only adds
// cold serialization work and memory that cannot be reused.
func buildFromFSOneShotCompactWithModuleCache(args []string, workDir string, stdRoot string, moduleCache string, fs SourceFS) BuildResult {
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
	if options.EmitUnit {
		// An emitted unit is a persistent interchange artifact. Preserve package
		// ownership and cache-key metadata so host and self-hosted frontends emit
		// the same canonical bytes.
		built = pipeline.BuildUnit(workDir, stdRoot, rootArg, sources.Files)
	} else {
		built = pipeline.BuildUnitWithTransientFiles(workDir, stdRoot, rootArg, sources.Files, sourcesStart, sourcesEnd)
	}
	result.Pipeline = built
	if !built.Ok {
		return buildFail(result, BuildErrPipeline, "", "", -1, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	result.Unit = built.Link.Data
	result.Sources = SourceResult{}
	return result
}

func buildFromFS(args []string, workDir string, stdRoot string, moduleCache string, fs SourceFS, compact bool) BuildResult {
	if compact {
		session := BeginFSBuildSession(args, workDir, stdRoot, moduleCache, fs, true)
		for !session.Step() {
		}
		return session.Result()
	}
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
	if compact && !options.EmitUnit {
		result.CacheKeyA, result.CacheKeyB = embeddedBuildFingerprint(workDir, options, sources.Files)
		if embeddedBuildCacheValid && result.CacheKeyA == embeddedBuildCacheKeyA && result.CacheKeyB == embeddedBuildCacheKeyB {
			if fs.PathExists(options.Output) {
				result.CacheHit = true
				result.Sources = SourceResult{}
				arena.Reset(sourcesStart)
				return result
			}
		}
	}
	rootArg := options.Package
	if len(options.Files) > 0 {
		rootArg = sources.Root.Dir
	}
	var built pipeline.Result
	if compact {
		built = pipeline.BuildUnitWithTransientFilesCached(workDir, stdRoot, rootArg, sources.Files, sourcesStart, sourcesEnd)
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

func rememberEmbeddedBuild(result BuildResult) {
	if result.CacheKeyA == 0 && result.CacheKeyB == 0 {
		return
	}
	embeddedBuildCacheKeyA = result.CacheKeyA
	embeddedBuildCacheKeyB = result.CacheKeyB
	embeddedBuildCacheValid = true
}

func embeddedBuildFingerprint(workDir string, options Options, files []load.SourceFile) (int, int) {
	a, b := 97, 193
	a, b = embeddedBuildHashString(a, b, workDir)
	a, b = embeddedBuildHashString(a, b, options.Target)
	a, b = embeddedBuildHashString(a, b, options.Output)
	a = embeddedBuildHashInt(a, options.ArenaSize)
	b = embeddedBuildHashIntB(b, options.ArenaSize)
	if options.Strip {
		a = embeddedBuildHashInt(a, 1)
		b = embeddedBuildHashIntB(b, 1)
	}
	if options.WindowsGUI {
		a = embeddedBuildHashInt(a, 2)
		b = embeddedBuildHashIntB(b, 2)
	}
	for i := 0; i < len(options.Tags); i++ {
		a, b = embeddedBuildHashString(a, b, options.Tags[i])
	}
	for i := 0; i < len(files); i++ {
		a, b = embeddedBuildHashString(a, b, files[i].Path)
		// Bundled standard-library and renvo.dev module sources are immutable
		// for the lifetime of this compiler process. Their paths and lengths
		// identify the embedded payload without rehashing megabytes on every
		// editor build.
		if !embeddedBuildImmutableSource(files[i].Path) {
			for j := 0; j < len(files[i].Src); j++ {
				a = embeddedBuildHashInt(a, int(files[i].Src[j]))
				b = embeddedBuildHashIntB(b, int(files[i].Src[j]))
			}
		}
		a = embeddedBuildHashInt(a, len(files[i].Src))
		b = embeddedBuildHashIntB(b, len(files[i].Src))
	}
	return a, b
}

func embeddedBuildImmutableSource(path string) bool {
	if !renvoBundledStdEnabled {
		return false
	}
	return embeddedBuildPathPrefix(path, "/std/") || embeddedBuildPathPrefix(path, "/modules/")
}

func embeddedBuildPathPrefix(path string, prefix string) bool {
	if len(path) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if path[i] != prefix[i] {
			return false
		}
	}
	return true
}

func embeddedBuildHashString(a int, b int, value string) (int, int) {
	for i := 0; i < len(value); i++ {
		a = embeddedBuildHashInt(a, int(value[i]))
		b = embeddedBuildHashIntB(b, int(value[i]))
	}
	return embeddedBuildHashInt(a, len(value)), embeddedBuildHashIntB(b, len(value))
}

func embeddedBuildHashInt(hash int, value int) int { return hash*131 + value + 1 }

func embeddedBuildHashIntB(hash int, value int) int { return hash*257 + value + 3 }

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
