package driver

import (
	"renvo.dev/internal/arena"
	"renvo.dev/internal/c11"
	"renvo.dev/internal/load"
)

type cObjectIncludeReader struct {
	fs    SourceFS
	paths []string
}

func (r cObjectIncludeReader) ReadInclude(from string, name string, angled bool) ([]byte, string, bool) {
	if !angled {
		path := load.JoinPath(load.DirPath(from), name)
		if src, ok := r.fs.ReadFile(path); ok {
			return src, path, true
		}
	}
	for i := 0; i < len(r.paths); i++ {
		path := load.JoinPath(r.paths[i], name)
		if src, ok := r.fs.ReadFile(path); ok {
			return src, path, true
		}
	}
	return nil, name, false
}

func (r cObjectIncludeReader) ReadIncludeNext(from string, name string, _ bool) ([]byte, string, bool) {
	fromDir := load.DirPath(from)
	start := 0
	for i := 0; i < len(r.paths); i++ {
		if load.CleanPath(r.paths[i]) == load.CleanPath(fromDir) {
			start = i + 1
			break
		}
	}
	for i := start; i < len(r.paths); i++ {
		path := load.JoinPath(r.paths[i], name)
		if src, ok := r.fs.ReadFile(path); ok {
			return src, path, true
		}
	}
	return nil, name, false
}

func prepareCObjectSources(result SourceResult, options *Options, workDir string, fs SourceFS) SourceResult {
	if !result.Ok || options.Mode != ModeObject {
		return result
	}
	reader := cObjectIncludeReader{fs: fs, paths: cObjectIncludePaths(workDir, options.IncludePaths, options.CNoStdIncludes, fs)}
	// Carry the target model explicitly so translation never depends on the host
	// Go process.
	dataModel := c11.DataModelLP64
	if options.Target == "linux/386" {
		dataModel = c11.DataModelILP32
	}
	for i := 0; i < len(result.Files); i++ {
		if !optionArgIsCFile(result.Files[i].Path) {
			continue
		}
		options.CDependencies = appendUniquePath(options.CDependencies, result.Files[i].Path)
		source := result.Files[i].Src
		headerSource := source
		if len(options.CForcedInclude) > 0 {
			prefix := make([]byte, 0, len(options.CForcedInclude)*32)
			for j := 0; j < len(options.CForcedInclude); j++ {
				path := load.JoinPath(workDir, options.CForcedInclude[j])
				prefix = append(prefix, "#include \""...)
				prefix = append(prefix, path...)
				prefix = append(prefix, '"', '\n')
			}
			prefix = append(prefix, source...)
			headerSource = prefix
		}
		header := c11.HeaderResult{Ok: true, ErrorAt: -1}
		if !options.CNoStdIncludes {
			header = c11.BuildObjectPrelude(result.Files[i].Path, headerSource, reader)
			if !header.Ok {
				result = sourceFail(result, SourceErrCInclude, header.ErrorPath)
				result.ErrorSourcePath = result.Files[i].Path
				result.ErrorOffset = header.ErrorAt
				return result
			}
		}
		processed := c11.Preprocess(c11.PreprocessConfig{
			Path: result.Files[i].Path, Source: source, Reader: reader,
			Predefined: cCommandMacros(*options), Undefined: cCommandUndefined(*options),
			ForcedIncludes: options.CForcedInclude, EmitIncludes: options.CNoStdIncludes, EmitQuotedIncludes: true,
			SuppressForcedIncludes: !options.CNoStdIncludes,
		})
		if !processed.Ok {
			result = sourceFail(result, SourceErrCPreprocess, processed.ErrorPath)
			result.CPreprocessError = processed.Error
			result.CPreprocessLine = processed.Line
			result.CPreprocessDetail = processed.Detail
			return result
		}
		result.Files[i].CObject = true
		result.Files[i].CDataModel = dataModel
		result.Files[i].CFunctionSections = options.CFunctionSections
		result.Files[i].CDataSections = options.CDataSections
		result.Files[i].CShortWChar = options.CShortWChar
		result.Files[i].CUnsignedChar = options.CUnsignedChar
		result.Files[i].CKernelCodeModel = options.CKernelCodeModel
		result.Files[i].COptimize = options.COptimize
		result.Files[i].CPrelude = header.Prelude
		result.Files[i].Src = processed.Source
		for j := 0; j < len(header.Dependencies); j++ {
			options.CDependencies = appendUniquePath(options.CDependencies, header.Dependencies[j])
		}
		for j := 0; j < len(processed.Dependencies); j++ {
			options.CDependencies = appendUniquePath(options.CDependencies, processed.Dependencies[j])
		}
	}
	return result
}

// CDependencyOutput emits the make-compatible dependency rule requested by a
// C compiler invocation. The dependency order follows source discovery and is
// deterministic for a fixed translation unit.
func CDependencyOutput(options Options) []byte {
	if len(options.CDependencyData) > 0 {
		return options.CDependencyData
	}
	if options.DependencyFile == "" {
		return nil
	}
	out := make([]byte, 0, CDependencyOutputSize(options))
	return AppendCDependencyOutput(out, options)
}

func CDependencyOutputSize(options Options) int {
	if len(options.CDependencyData) > 0 {
		return len(options.CDependencyData)
	}
	if options.DependencyFile == "" {
		return 0
	}
	target := options.DependencyTarget
	if target == "" {
		target = options.Output
	}
	size := makeDependencyWordSize(target) + 2
	for i := 0; i < len(options.CDependencies); i++ {
		size += 1 + makeDependencyWordSize(cDependencyPath(options, options.CDependencies[i]))
	}
	return size
}

func AppendCDependencyOutput(out []byte, options Options) []byte {
	if len(options.CDependencyData) > 0 {
		return append(out, options.CDependencyData...)
	}
	if options.DependencyFile == "" {
		return out
	}
	target := options.DependencyTarget
	if target == "" {
		target = options.Output
	}
	out = appendMakeDependencyWord(out, target)
	out = append(out, ':')
	for i := 0; i < len(options.CDependencies); i++ {
		out = append(out, ' ')
		out = appendMakeDependencyWord(out, cDependencyPath(options, options.CDependencies[i]))
	}
	out = append(out, '\n')
	return out
}

func finalizeCDependencyOptions(options Options) Options {
	if options.DependencyFile == "" && options.CDependencyRequested {
		options.DependencyFile = defaultCDependencyPath(options.Output)
	}
	if options.DependencyFile == "" || len(options.CDependencyData) > 0 {
		return options
	}
	options.CDependencyData = arena.PersistBytes(CDependencyOutput(options))
	return options
}

func defaultCDependencyPath(output string) string {
	if len(output) > 2 && output[len(output)-2:] == ".o" {
		return output[:len(output)-1] + "d"
	}
	return output + ".d"
}

func cDependencyPath(options Options, path string) string {
	root := options.CDependencyRoot
	if root == "" || root == "/" || len(path) <= len(root) || path[:len(root)] != root || path[len(root)] != '/' {
		return path
	}
	return path[len(root)+1:]
}

func makeDependencyWordSize(value string) int {
	size := len(value)
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case ' ', '\t', '#', ':', '$':
			size++
		}
	}
	return size
}

func appendMakeDependencyWord(out []byte, value string) []byte {
	for i := 0; i < len(value); i++ {
		switch value[i] {
		case ' ', '\t', '#', ':':
			out = append(out, '\\')
		case '$':
			out = append(out, '$')
		}
		out = append(out, value[i])
	}
	return out
}

func cObjectIncludePaths(workDir string, explicit []string, noStandard bool, fs SourceFS) []string {
	paths := make([]string, 0, len(explicit)+8)
	for i := 0; i < len(explicit); i++ {
		path := load.JoinPath(workDir, explicit[i])
		paths = appendUniquePath(paths, path)
	}
	if noStandard {
		return paths
	}

	// GCC installs target-independent headers such as stddef.h ahead of the
	// libc include roots. Discover the active x86_64 directory without invoking
	// another compiler or baking a compiler version into Renvo.
	targets, ok := fs.ReadDir("/usr/lib/gcc")
	if ok {
		sortDirEntries(targets)
		for i := len(targets) - 1; i >= 0; i-- {
			if !targets[i].IsDir || !textContains(targets[i].Name, "x86_64") {
				continue
			}
			targetRoot := load.JoinPath("/usr/lib/gcc", targets[i].Name)
			versions, readable := fs.ReadDir(targetRoot)
			if !readable {
				continue
			}
			sortDirEntries(versions)
			for j := len(versions) - 1; j >= 0; j-- {
				if !versions[j].IsDir {
					continue
				}
				include := load.JoinPath(load.JoinPath(targetRoot, versions[j].Name), "include")
				if fs.PathExists(include) {
					paths = appendUniquePath(paths, include)
				}
			}
		}
	}
	if fs.PathExists("/usr/local/include") {
		paths = appendUniquePath(paths, "/usr/local/include")
	}
	if fs.PathExists("/usr/include/x86_64-linux-gnu") {
		paths = appendUniquePath(paths, "/usr/include/x86_64-linux-gnu")
	}
	if fs.PathExists("/usr/include") {
		paths = appendUniquePath(paths, "/usr/include")
	}
	return paths
}

func appendUniquePath(paths []string, path string) []string {
	path = load.CleanPath(path)
	for i := 0; i < len(paths); i++ {
		if paths[i] == path {
			return paths
		}
	}
	return append(paths, path)
}

func textContains(value string, part string) bool {
	if len(part) == 0 {
		return true
	}
	for i := 0; i+len(part) <= len(value); i++ {
		matched := true
		for j := 0; j < len(part); j++ {
			if value[i+j] != part[j] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}
