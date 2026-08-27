package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/pipeline"
	"renvo.dev/internal/unit"
)

// FSBuildSession advances source discovery and the package pipeline in bounded
// steps. It is used by embedded GUI callers that must return to an event loop.
type FSBuildSession struct {
	args            []string
	workDir         string
	stdRoot         string
	moduleCache     string
	fs              SourceFS
	compact         bool
	cached          bool
	stage           int
	sourcesStart    int
	sourcesEnd      int
	rootArg         string
	pipeline        *pipeline.Session
	targetBinding   unit.TargetBinding
	foreignPrograms []unit.ForeignProgram
	result          BuildResult
}

func BeginFSBuildSession(args []string, workDir string, stdRoot string, moduleCache string, fs SourceFS, compact bool) *FSBuildSession {
	return beginFSBuildSession(args, workDir, stdRoot, moduleCache, fs, compact, compact)
}

func beginFSBuildSession(args []string, workDir string, stdRoot string, moduleCache string, fs SourceFS, compact bool, cached bool) *FSBuildSession {
	return &FSBuildSession{
		args:        args,
		workDir:     workDir,
		stdRoot:     stdRoot,
		moduleCache: moduleCache,
		fs:          fs,
		compact:     compact,
		cached:      cached,
		result:      newBuildResult(),
	}
}

func (s *FSBuildSession) Step() bool {
	if s == nil || s.stage >= 4 {
		return true
	}
	if s.stage == 0 {
		options, binding := resolveFSBuildSessionOptions(s.args, s.workDir, s.fs)
		s.targetBinding = binding
		s.result.Options = options
		if !options.Ok {
			s.result = buildFail(s.result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
			s.stage = 4
			return true
		}
		options = resolveCCompilerPaths(s.workDir, options)
		s.workDir, options = standaloneCSourceContext(s.workDir, options)
		s.result.Options = options
		s.fs = sourceFSForOptions(s.fs, s.workDir, options)
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
		sources = prepareCSources(sources, &options, s.workDir, s.stdRoot, s.moduleCache, s.fs)
		options = finalizeCDependencyOptions(options)
		s.sourcesEnd = arena.Mark()
		s.result.Sources = sources
		s.result.Options = options
		if !sources.Ok {
			s.result = buildFail(s.result, BuildErrSource, "", sources.ErrorPath, -1, -1, -1, -1)
			s.stage = 4
			return true
		}
		options = resolveModuleLicense(options, sources.Files)
		s.result.Options = options
		if !options.Ok {
			s.result = buildFail(s.result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
			s.stage = 4
			return true
		}
		if s.cached && !options.EmitUnit {
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
		foreign := prepareForeignPrograms(options, s.workDir, s.stdRoot, s.moduleCache, sources, s.fs)
		if !foreign.Ok {
			s.result = buildForeignFail(s.result, foreign.Diagnostic)
			s.stage = 4
			return true
		}
		s.foreignPrograms = foreign.Programs
		if options.Mode == ModeObject {
			// Object linking rewrites source-token line fields while producing the
			// compact mapping, so its source arena cannot yet be transient.
			s.pipeline = pipeline.BeginObjectSession(
				s.workDir, s.stdRoot, s.rootArg, sources.Files, s.cached)
		} else {
			s.pipeline = pipeline.BeginSession(
				s.workDir, s.stdRoot, s.rootArg, sources.Files,
				s.sourcesStart, s.sourcesEnd, s.compact, s.cached)
		}
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
		if !bindForeignPrograms(&s.result.Unit, s.foreignPrograms) {
			s.result = buildForeignFail(s.result, Diagnostic{Phase: "foreign", Code: "RENVO-FOREIGN-008", Message: "could not encode foreign units"})
			s.stage = 4
			return true
		}
		bindBuiltInTarget(&s.result.Unit, s.result.Options)
		if s.targetBinding.Target != "" {
			if bound, ok := unit.BindTarget(s.result.Unit, s.targetBinding); ok {
				s.result.Unit = bound
			}
		}
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
