package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/pipeline"
)

// FSBuildSession advances source discovery and the package pipeline in bounded
// steps. It is used by embedded GUI callers that must return to an event loop.
type FSBuildSession struct {
	args         []string
	workDir      string
	stdRoot      string
	moduleCache  string
	fs           SourceFS
	compact      bool
	stage        int
	sourcesStart int
	sourcesEnd   int
	rootArg      string
	pipeline     *pipeline.Session
	result       BuildResult
}

func BeginFSBuildSession(args []string, workDir string, stdRoot string, moduleCache string, fs SourceFS, compact bool) *FSBuildSession {
	return &FSBuildSession{
		args:        args,
		workDir:     workDir,
		stdRoot:     stdRoot,
		moduleCache: moduleCache,
		fs:          fs,
		compact:     compact,
		result:      newBuildResult(),
	}
}

func (s *FSBuildSession) Step() bool {
	if s == nil || s.stage >= 4 {
		return true
	}
	if s.stage == 0 {
		options := ParseOptions(s.args)
		s.result.Options = options
		if !options.Ok {
			s.result = buildFail(s.result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
			s.stage = 4
			return true
		}
		s.stage = 1
		return false
	}
	if s.stage == 1 {
		s.sourcesStart = arena.Mark()
		options := s.result.Options
		var sources SourceResult
		if len(options.Files) > 0 {
			sources = CollectSourceFilesForTargetTagsWithModuleCache(s.workDir, s.stdRoot, options.Files, options.Target, options.Tags, s.moduleCache, s.fs)
		} else {
			sources = CollectSourcesForTargetTagsWithModuleCache(s.workDir, s.stdRoot, options.Package, options.Target, options.Tags, s.moduleCache, s.fs)
		}
		s.sourcesEnd = arena.Mark()
		s.result.Sources = sources
		if !sources.Ok {
			s.result = buildFail(s.result, BuildErrSource, "", sources.ErrorPath, -1, -1, -1, -1)
			s.stage = 4
			return true
		}
		if s.compact && !options.EmitUnit {
			s.result.CacheKeyA, s.result.CacheKeyB = embeddedBuildFingerprint(s.workDir, options, sources.Files)
			if embeddedBuildCacheValid && s.result.CacheKeyA == embeddedBuildCacheKeyA && s.result.CacheKeyB == embeddedBuildCacheKeyB && s.fs.PathExists(options.Output) {
				s.result.CacheHit = true
				s.result.Sources = SourceResult{}
				arena.Reset(s.sourcesStart)
				s.stage = 4
				return true
			}
		}
		s.rootArg = options.Package
		if len(options.Files) > 0 {
			s.rootArg = sources.Root.Dir
		}
		s.pipeline = pipeline.BeginSession(s.workDir, s.stdRoot, s.rootArg, sources.Files, s.sourcesStart, s.sourcesEnd, s.compact, s.compact)
		s.stage = 2
		return false
	}
	if s.stage == 2 {
		if !s.pipeline.Step() {
			return false
		}
		built := s.pipeline.Result()
		s.result.Pipeline = built
		if !built.Ok {
			s.result = buildFail(s.result, BuildErrPipeline, "", "", -1, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
			s.stage = 4
			return true
		}
		s.result.Unit = built.Link.Data
		s.stage = 3
		return false
	}
	if s.compact {
		s.result.Sources = SourceResult{}
	}
	s.stage = 4
	return true
}

func (s *FSBuildSession) Result() BuildResult {
	if s == nil {
		return buildFail(newBuildResult(), BuildErrSource, "", "", -1, -1, -1, -1)
	}
	return s.result
}
