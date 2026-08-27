package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/load"
	"renvo.dev/internal/pipeline"
	"renvo.dev/internal/targetinfo"
	"renvo.dev/internal/unit"
)

const (
	BuildOK = iota
	BuildErrOptions
	BuildErrSource
	BuildErrPipeline
	BuildErrForeign
)

type BuildResult struct {
	Options           Options
	Sources           SourceResult
	Pipeline          pipeline.Result
	Unit              []byte
	Ok                bool
	Error             int
	ErrorArg          string
	ErrorPath         string
	ErrorAt           int
	ErrorPackage      int
	ErrorFile         int
	ErrorToken        int
	Diagnostic        Diagnostic
	ForeignDiagnostic Diagnostic
	CacheKeyA         int
	CacheKeyB         int
	CacheHit          bool
}

var embeddedBuildCacheValid bool
var embeddedBuildCacheKeyA int
var embeddedBuildCacheKeyB int

func BuildUnit(args []string, workDir string, stdRoot string, files []load.SourceFile) BuildResult {
	result := newBuildResult()
	options := ParseOptions(args)
	if options.Ok && options.System != "" {
		options.SystemError = "-system requires a filesystem-backed build"
		options = parseFail(options, ParseErrSystemRead, options.System, options.SystemAt)
	}
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
	}
	filtered, errorPath, sourceError := filterSourcesForOptions(files, workDir, options)
	if sourceError != SourceOK {
		result.Sources = SourceResult{Error: sourceError, ErrorPath: errorPath}
		return buildFail(result, BuildErrSource, "", errorPath, -1, -1, -1, -1)
	}
	filtered = transformScriptFiles(filtered, workDir, options)
	options = resolveModuleLicense(options, filtered)
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
	}
	rootArg := options.Package
	if len(options.Files) > 0 {
		rootArg = load.DirPath(load.JoinPath(workDir, options.Files[0]))
	}
	foreignRoot := load.JoinPath(workDir, rootArg)
	foreignSources := SourceResult{Files: filtered, Root: load.PackageRef{Dir: foreignRoot}, Ok: true}
	foreign := prepareForeignPrograms(options, workDir, stdRoot, "", foreignSources, nil)
	if !foreign.Ok {
		return buildForeignFail(result, foreign.Diagnostic)
	}
	built := pipeline.BuildUnit(workDir, stdRoot, rootArg, filtered)
	result.Pipeline = built
	if !built.Ok {
		return buildFail(result, BuildErrPipeline, "", built.ErrorPath, built.ErrorOffset, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	result.Unit = built.Link.Data
	if !bindForeignPrograms(&result.Unit, built.Link.Program, foreign.Programs) {
		return buildForeignFail(result, Diagnostic{Phase: "foreign", Code: "RENVO-FOREIGN-008", Message: "could not encode foreign units"})
	}
	bindBuiltInTarget(&result.Unit, options)
	return result
}

func BuildFromFS(args []string, workDir string, stdRoot string, fs SourceFS) BuildResult {
	return buildFromFS(args, workDir, stdRoot, "", fs, false)
}

// BuildFromFSOneShot builds one command-line invocation without retaining the
// incremental session state used by editors and long-lived compiler hosts.
func BuildFromFSOneShot(args []string, workDir string, stdRoot string, fs SourceFS) BuildResult {
	return buildFromFSOneShotCompactWithModuleCache(args, workDir, stdRoot, "", fs)
}

// BuildPackageUnitFromFS is the narrow frontend boundary used by fixed-target
// command-line compilers. Option parsing and backend selection stay in the
// host, while this function only discovers, checks, and lowers one package.
func BuildPackageUnitFromFS(packageArg string, target string, tags []string, workDir string, stdRoot string, fs SourceFS) BuildResult {
	result := newBuildResult()
	result.Options = Options{Package: packageArg, Target: target, Tags: tags, Ok: true}
	sources := CollectSourcesForTargetTagsWithModuleCache(workDir, stdRoot, packageArg, target, tags, "", fs)
	result.Sources = sources
	if !sources.Ok {
		return buildFail(result, BuildErrSource, "", sources.ErrorPath, -1, -1, -1, -1)
	}
	foreign := prepareForeignPrograms(result.Options, workDir, stdRoot, "", sources, fs)
	if !foreign.Ok {
		return buildForeignFail(result, foreign.Diagnostic)
	}
	built := pipeline.BuildUnit(workDir, stdRoot, packageArg, sources.Files)
	result.Pipeline = built
	if !built.Ok {
		return buildFail(result, BuildErrPipeline, "", built.ErrorPath, built.ErrorOffset, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	result.Unit = built.Link.Data
	if !bindForeignPrograms(&result.Unit, built.Link.Program, foreign.Programs) {
		return buildForeignFail(result, Diagnostic{Phase: "foreign", Code: "RENVO-FOREIGN-008", Message: "could not encode foreign units"})
	}
	bindBuiltInTarget(&result.Unit, result.Options)
	return result
}

type PackageUnitResult struct {
	Unit         []byte
	Path         string
	Ok           bool
	Phase        int
	Error        int
	ErrorPackage int
	ErrorFile    int
	ErrorToken   int
	Diagnostic   Diagnostic
}

// BuildPackageUnitCompact preserves the shared structured diagnostic while
// leaving its presentation to the command or browser host.
func BuildPackageUnitCompact(packageArg string, target string, tags []string, workDir string, stdRoot string, fs SourceFS) PackageUnitResult {
	return BuildPackageUnitCompactMode(packageArg, target, tags, ModeExecutable, workDir, stdRoot, fs)
}

// BuildPackageUnitCompactMode is the fixed-target frontend boundary with the
// mode distinctions required by object and kernel-module browser builds.
func BuildPackageUnitCompactMode(packageArg string, target string, tags []string, mode string, workDir string, stdRoot string, fs SourceFS) PackageUnitResult {
	var result PackageUnitResult
	options := Options{Package: packageArg, Target: target, Tags: tags, Mode: mode, Ok: true}
	sources := CollectSourcesForTargetTagsWithModuleCache(workDir, stdRoot, packageArg, target, tags, "", fs)
	if !sources.Ok {
		result.Phase = BuildErrSource
		result.Error = sources.Error
		result.Path = sources.ErrorPath
		failed := BuildResult{Sources: sources, Error: BuildErrSource, ErrorPath: sources.ErrorPath,
			ErrorAt: -1, ErrorPackage: -1, ErrorFile: -1, ErrorToken: -1}
		result.Diagnostic = diagnosticForBuild(failed)
		return result
	}
	foreign := prepareForeignPrograms(options, workDir, stdRoot, "", sources, fs)
	if !foreign.Ok {
		result.Phase = BuildErrForeign
		result.Diagnostic = foreign.Diagnostic
		return result
	}
	var built pipeline.Result
	if mode == ModeObject {
		built = pipeline.BuildObjectUnit(workDir, stdRoot, packageArg, sources.Files)
	} else {
		built = pipeline.BuildUnit(workDir, stdRoot, packageArg, sources.Files)
	}
	if !built.Ok {
		result.Phase = BuildErrPipeline
		result.Error = built.Error
		result.Path = built.ErrorPath
		result.ErrorPackage = built.ErrorPackage
		result.ErrorFile = built.ErrorFile
		result.ErrorToken = built.ErrorToken
		failed := BuildResult{Sources: sources, Pipeline: built, Error: BuildErrPipeline,
			ErrorPath: built.ErrorPath, ErrorAt: built.ErrorOffset, ErrorPackage: built.ErrorPackage,
			ErrorFile: built.ErrorFile, ErrorToken: built.ErrorToken}
		result.Diagnostic = diagnosticForBuild(failed)
		return result
	}
	result.Unit = built.Link.Data
	if !bindForeignPrograms(&result.Unit, built.Link.Program, foreign.Programs) {
		result.Phase = BuildErrForeign
		result.Diagnostic = Diagnostic{Phase: "foreign", Code: "RENVO-FOREIGN-008", Message: "could not encode foreign units"}
		return result
	}
	bindBuiltInTarget(&result.Unit, options)
	result.Ok = true
	return result
}

// BuildCExecutableUnitCompact builds explicitly named, module-free C sources
// for the compact browser frontend. C dependency sources are synthesized while
// loading, so the persistent linker keeps their storage alive through unit
// serialization; the browser process exits after this one build.
func BuildCExecutableUnitCompact(files []string, target string, tags []string, workDir string, stdRoot string, fs SourceFS) PackageUnitResult {
	if len(files) == 0 {
		return PackageUnitResult{Phase: BuildErrSource, Error: SourceErrFileListEmpty}
	}
	options := Options{
		Target: target, Mode: ModeExecutable, ModuleLicense: DefaultModuleLicense,
		Files: files, Package: files[0], CCompiler: true, Ok: true, ErrorAt: -1, SystemAt: -1,
		Tags: tags,
	}
	built := buildFromFSOptions(options, workDir, stdRoot, "", fs, false)
	result := PackageUnitResult{
		Unit: built.Unit, Path: built.ErrorPath, Ok: built.Ok, Phase: built.Error,
		ErrorPackage: built.ErrorPackage, ErrorFile: built.ErrorFile, ErrorToken: built.ErrorToken,
		Diagnostic: built.Diagnostic,
	}
	if built.Ok {
		result.Phase = BuildOK
		result.Error = SourceOK
		return result
	}
	result.Phase = built.Error
	if built.Error == BuildErrSource {
		result.Error = built.Sources.Error
	} else if built.Error == BuildErrPipeline {
		result.Error = built.Pipeline.Error
		if built.Pipeline.Error == pipeline.PipelineErrLoad {
			result.Error = built.Pipeline.Workspace.Error
		}
	}
	return result
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
	options := parseFSOptions(args, workDir, fs)
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
	}
	options = resolveCCompilerPaths(workDir, options)
	workDir, options = standaloneCSourceContext(workDir, options)
	result.Options = options
	fs = sourceFSForOptions(fs, workDir, options)
	sourcesStart := arena.Mark()
	var sources SourceResult
	if len(options.Files) > 0 {
		sources = CollectSourceFilesForTargetTagsWithModuleCache(workDir, stdRoot, options.Files, options.Target, options.Tags, moduleCache, fs)
	} else {
		sources = CollectSourcesForTargetTagsWithModuleCache(workDir, stdRoot, options.Package, options.Target, options.Tags, moduleCache, fs)
	}
	sources = prepareCSources(sources, &options, workDir, stdRoot, moduleCache, fs)
	options = finalizeCDependencyOptions(options)
	sourcesEnd := arena.Mark()
	result.Sources = sources
	result.Options = options
	if !sources.Ok {
		return buildFail(result, BuildErrSource, "", sources.ErrorPath, -1, -1, -1, -1)
	}
	options = resolveModuleLicense(options, sources.Files)
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
	}
	rootArg := options.Package
	if len(options.Files) > 0 {
		rootArg = sources.Root.Dir
	}
	foreign := prepareForeignPrograms(options, workDir, stdRoot, moduleCache, sources, fs)
	if !foreign.Ok {
		return buildForeignFail(result, foreign.Diagnostic)
	}
	var built pipeline.Result
	if options.Mode == ModeObject && options.EmitUnit {
		built = pipeline.BuildObjectUnit(workDir, stdRoot, rootArg, sources.Files)
	} else if options.Mode == ModeObject {
		// Object linking rewrites source-token line fields as a compact mapping.
		// Keep the source package resident until that mapping has been restored;
		// retiring its arena pages during the link can alias a small destination
		// unit and turn token indexes into source lines before backend decoding.
		built = pipeline.BuildObjectUnit(workDir, stdRoot, rootArg, sources.Files)
	} else if options.EmitUnit || len(foreign.Programs) > 0 {
		// An emitted unit is a persistent interchange artifact. Preserve package
		// ownership and cache-key metadata so host and self-hosted frontends emit
		// the same canonical bytes.
		built = pipeline.BuildUnit(workDir, stdRoot, rootArg, sources.Files)
	} else {
		built = pipeline.BuildUnitWithTransientFiles(workDir, stdRoot, rootArg, sources.Files, sourcesStart, sourcesEnd)
	}
	result.Pipeline = built
	if !built.Ok {
		return buildFail(result, BuildErrPipeline, "", built.ErrorPath, built.ErrorOffset, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	result.Unit = built.Link.Data
	if !bindForeignPrograms(&result.Unit, built.Link.Program, foreign.Programs) {
		return buildForeignFail(result, Diagnostic{Phase: "foreign", Code: "RENVO-FOREIGN-008", Message: "could not encode foreign units"})
	}
	bindBuiltInTarget(&result.Unit, options)
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
	options := parseFSOptions(args, workDir, fs)
	return buildFromFSOptions(options, workDir, stdRoot, moduleCache, fs, compact)
}

func buildFromFSOptions(options Options, workDir string, stdRoot string, moduleCache string, fs SourceFS, compact bool) BuildResult {
	result := newBuildResult()
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
	}
	options = resolveCCompilerPaths(workDir, options)
	workDir, options = standaloneCSourceContext(workDir, options)
	result.Options = options
	fs = sourceFSForOptions(fs, workDir, options)
	sourcesStart := arena.Mark()
	var sources SourceResult
	if len(options.Files) > 0 {
		sources = CollectSourceFilesForTargetTagsWithModuleCache(workDir, stdRoot, options.Files, options.Target, options.Tags, moduleCache, fs)
	} else {
		sources = CollectSourcesForTargetTagsWithModuleCache(workDir, stdRoot, options.Package, options.Target, options.Tags, moduleCache, fs)
	}
	sources = prepareCSources(sources, &options, workDir, stdRoot, moduleCache, fs)
	options = finalizeCDependencyOptions(options)
	sourcesEnd := arena.Mark()
	result.Sources = sources
	result.Options = options
	if !sources.Ok {
		return buildFail(result, BuildErrSource, "", sources.ErrorPath, -1, -1, -1, -1)
	}
	options = resolveModuleLicense(options, sources.Files)
	result.Options = options
	if !options.Ok {
		return buildFail(result, BuildErrOptions, options.ErrorArg, "", options.ErrorAt, -1, -1, -1)
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
	foreign := prepareForeignPrograms(options, workDir, stdRoot, moduleCache, sources, fs)
	if !foreign.Ok {
		return buildForeignFail(result, foreign.Diagnostic)
	}
	var built pipeline.Result
	if compact && options.Mode == ModeObject {
		// Object linking temporarily rewrites source-token line fields. Keep the
		// source package resident until it restores them; the ordinary transient
		// pipeline can otherwise recycle that storage into the destination unit.
		built = pipeline.BuildObjectUnit(workDir, stdRoot, rootArg, sources.Files)
	} else if options.Mode == ModeObject {
		built = pipeline.BuildObjectUnit(workDir, stdRoot, rootArg, sources.Files)
	} else if compact && len(foreign.Programs) > 0 {
		built = pipeline.BuildUnit(workDir, stdRoot, rootArg, sources.Files)
	} else if compact && options.CCompiler {
		// Demand-selected libc sources are synthesized during collection rather
		// than retained by the editor cache. Use the one-shot transient linker so
		// their storage remains valid through serialization without being pinned
		// for the next incremental build.
		built = pipeline.BuildUnitWithTransientFiles(workDir, stdRoot, rootArg, sources.Files, sourcesStart, sourcesEnd)
	} else if compact {
		built = pipeline.BuildUnitWithTransientFilesCached(workDir, stdRoot, rootArg, sources.Files, sourcesStart, sourcesEnd)
	} else {
		built = pipeline.BuildUnit(workDir, stdRoot, rootArg, sources.Files)
	}
	result.Pipeline = built
	if !built.Ok {
		return buildFail(result, BuildErrPipeline, "", built.ErrorPath, built.ErrorOffset, built.ErrorPackage, built.ErrorFile, built.ErrorToken)
	}
	result.Unit = built.Link.Data
	if !bindForeignPrograms(&result.Unit, built.Link.Program, foreign.Programs) {
		return buildForeignFail(result, Diagnostic{Phase: "foreign", Code: "RENVO-FOREIGN-008", Message: "could not encode foreign units"})
	}
	bindBuiltInTarget(&result.Unit, options)
	if compact {
		result.Sources = SourceResult{}
	}
	return result
}

func bindBuiltInTarget(data *[]byte, options Options) {
	name := backendTargetForOptions(options.Target, options.Mode)
	target, definition, version, ok := targetinfo.Binding(name)
	if !ok {
		target, definition, version, ok = renvoBackendTargetBinding(name)
	}
	if !ok {
		return
	}
	var targetBinding unit.TargetBinding
	targetBinding.Target = target
	targetBinding.Definition = definition
	targetBinding.DescriptorVersion = version
	unit.BindUnboundTarget(data, targetBinding)
}

func bindForeignPrograms(data *[]byte, program unit.Program, programs []unit.ForeignProgram) bool {
	for i := 0; i < len(programs); i++ {
		programs[i].Global = linkedForeignGlobal(program, programs[i].Name)
		if programs[i].Global < 0 {
			return false
		}
	}
	bound, ok := unit.BindForeignPrograms(*data, programs)
	if ok {
		*data = bound
	}
	return ok
}

func rememberEmbeddedBuild(result BuildResult) {
	if result.CacheKeyA == 0 && result.CacheKeyB == 0 {
		return
	}
	embeddedBuildCacheKeyA = result.CacheKeyA
	embeddedBuildCacheKeyB = result.CacheKeyB
	embeddedBuildCacheValid = true
}

// RememberBuildOutput marks a successfully written build output as current.
// A later cached FSBuildSession can then skip both frontend and backend work
// when its source fingerprint and output path are unchanged.
func RememberBuildOutput(result BuildResult) {
	rememberEmbeddedBuild(result)
}

func embeddedBuildFingerprint(workDir string, options Options, files []load.SourceFile) (int, int) {
	a, b := 97, 193
	a, b = embeddedBuildHashString(a, b, workDir)
	a, b = embeddedBuildHashString(a, b, options.Target)
	a, b = embeddedBuildHashString(a, b, options.Output)
	a, b = embeddedBuildHashString(a, b, options.ModuleLicense)
	a, b = embeddedBuildHashString(a, b, options.SystemName)
	a = embeddedBuildHashInt(a, options.ArenaSize)
	b = embeddedBuildHashIntB(b, options.ArenaSize)
	a = embeddedBuildHashInt(a, options.BinaryLimit)
	b = embeddedBuildHashIntB(b, options.BinaryLimit)
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

func resolveModuleLicense(options Options, files []load.SourceFile) Options {
	if options.Mode != ModeKernelModule {
		return options
	}
	license := ""
	for i := 0; i < len(files); i++ {
		value, found, valid, conflict := sourceModuleLicense(files[i].Src)
		if conflict {
			return parseFail(options, ParseErrConflictingModuleLicense, files[i].Path, -1)
		}
		if !valid {
			return parseFail(options, ParseErrInvalidModuleLicense, files[i].Path, -1)
		}
		if !found {
			continue
		}
		if license != "" && license != value {
			return parseFail(options, ParseErrConflictingModuleLicense, files[i].Path, -1)
		}
		license = value
	}
	if license != "" {
		options.ModuleLicense = license
	}
	return options
}

func sourceModuleLicense(src []byte) (string, bool, bool, bool) {
	prefix := "// renvo:module-license "
	license := ""
	found := false
	for start := 0; start < len(src); {
		end := start
		for end < len(src) && src[end] != '\n' {
			end++
		}
		lineStart := start
		for lineStart < end && (src[lineStart] == ' ' || src[lineStart] == '\t') {
			lineStart++
		}
		if end-lineStart >= len(prefix) {
			match := true
			for i := 0; i < len(prefix); i++ {
				if src[lineStart+i] != prefix[i] {
					match = false
				}
			}
			if match {
				valueStart := lineStart + len(prefix)
				valueEnd := end
				for valueEnd > valueStart && (src[valueEnd-1] == ' ' || src[valueEnd-1] == '\t' || src[valueEnd-1] == '\r') {
					valueEnd--
				}
				if valueEnd == valueStart || valueEnd-valueStart > 64 {
					return "", true, false, false
				}
				for i := valueStart; i < valueEnd; i++ {
					if src[i] < 32 || src[i] > 126 {
						return "", true, false, false
					}
				}
				value := string(src[valueStart:valueEnd])
				if found && value != license {
					return "", true, true, true
				}
				license, found = value, true
			}
		}
		start = end + 1
	}
	return license, found, true, false
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

func buildForeignFail(result BuildResult, diagnostic Diagnostic) BuildResult {
	result.ForeignDiagnostic = diagnostic
	return buildFail(result, BuildErrForeign, "", diagnostic.Path, diagnostic.Start, -1, -1, -1)
}
