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

func prepareCSources(result SourceResult, options *Options, workDir string, stdRoot string, moduleCache string, fs SourceFS) SourceResult {
	return prepareCSourcesPass(result, options, workDir, stdRoot, moduleCache, fs, true)
}

func prepareCSourcesPass(result SourceResult, options *Options, workDir string, stdRoot string, moduleCache string, fs SourceFS, reloadGoFiles bool) SourceResult {
	object := options.Mode == ModeObject
	executable := options.CCompiler && options.Mode == ModeExecutable
	if !result.Ok || !object && !executable {
		return result
	}
	paths := options.IncludePaths
	if object {
		paths = cObjectIncludePaths(workDir, paths, options.CNoStdIncludes, fs)
	} else if !options.CNoStdIncludes {
		copied := make([]string, len(paths), len(paths)+1)
		copy(copied, paths)
		paths = append(copied, load.JoinPath(load.DirPath(stdRoot), "libc/include"))
	}
	reader := cObjectIncludeReader{fs: fs, paths: paths}
	libcRoot := load.JoinPath(load.DirPath(stdRoot), "libc")
	selectedLibc := make([]string, 0, 5)
	pragmaGoFiles := make([]string, 0, 2)
	needsRuntime := false
	firstC := -1
	// Carry the target model explicitly so translation never depends on the host
	// Go process.
	dataModel := c11.DataModelLP64
	targetOS, _, pointerBits := cCompilerTarget(options.Target)
	if pointerBits == 32 {
		dataModel = c11.DataModelILP32
	} else if targetOS == "windows" {
		dataModel = c11.DataModelLLP64
	}
	for i := 0; i < len(result.Files); i++ {
		if !optionArgIsCFile(result.Files[i].Path) {
			continue
		}
		if firstC < 0 {
			firstC = i
		}
		options.CDependencies = appendUniquePath(options.CDependencies, result.Files[i].Path)
		source := result.Files[i].Src
		processed := c11.Preprocess(c11.PreprocessConfig{
			Path: result.Files[i].Path, Source: source, Reader: reader,
			Predefined: cCommandMacros(*options), Undefined: cCommandUndefined(*options),
			ForcedIncludes: options.CForcedInclude, EmitIncludes: executable || options.CNoStdIncludes, EmitQuotedIncludes: true,
			SuppressForcedIncludes: object && !options.CNoStdIncludes,
		})
		if !processed.Ok {
			errorKind := SourceErrCPreprocess
			if processed.Error == c11.PreprocessErrInclude {
				errorKind = SourceErrCInclude
			}
			result = sourceFail(result, errorKind, processed.ErrorPath)
			result.ErrorSourcePath = result.Files[i].Path
			result.ErrorOffset = 0
			result.CPreprocessError = processed.Error
			result.CPreprocessLine = processed.Line
			result.CPreprocessDetail = processed.Detail
			return result
		}
		header := c11.HeaderResult{Ok: true, ErrorAt: -1}
		if object && !options.CNoStdIncludes {
			headers := make([]c11.HeaderSource, 0, len(processed.Dependencies))
			for j := 0; j < len(processed.Dependencies); j++ {
				src, read := fs.ReadFile(processed.Dependencies[j])
				if !read {
					result = sourceFail(result, SourceErrCInclude, processed.Dependencies[j])
					result.ErrorSourcePath = result.Files[i].Path
					return result
				}
				headers = append(headers, c11.HeaderSource{Path: processed.Dependencies[j], Source: src})
			}
			header = c11.BuildObjectPrelude(result.Files[i].Path, processed.Source, reader, headers...)
			if !header.Ok {
				result = sourceFail(result, SourceErrCInclude, header.ErrorPath)
				result.ErrorSourcePath = result.Files[i].Path
				result.ErrorOffset = header.ErrorAt
				return result
			}
		}
		result.Files[i].CObject = object
		result.Files[i].CCompiler = options.CCompiler
		result.Files[i].CDataModel = dataModel
		result.Files[i].CFunctionSections = options.CFunctionSections
		result.Files[i].CDataSections = options.CDataSections
		result.Files[i].CShortWChar = options.CShortWChar
		result.Files[i].CUnsignedChar = options.CUnsignedChar
		result.Files[i].CKernelCodeModel = options.CKernelCodeModel
		result.Files[i].COptimize = options.COptimize
		for j := 0; j < len(processed.GoFiles); j++ {
			path := load.CleanPath(load.JoinPath(load.DirPath(processed.GoFiles[j].From), processed.GoFiles[j].Name))
			if !isGoSourceName(load.BasePath(path)) {
				return sourceFail(result, SourceErrReadFile, path)
			}
			if load.DirPath(path) != load.CleanPath(workDir) {
				return sourceFail(result, SourceErrFileDirectory, path)
			}
			name := load.BasePath(path)
			if findString(pragmaGoFiles, name) < 0 {
				pragmaGoFiles = append(pragmaGoFiles, name)
				options.CDependencies = appendUniquePath(options.CDependencies, path)
			}
		}
		result.Files[i].CPrelude = header.Prelude
		result.Files[i].Src = processed.Source
		for j := 0; j < len(header.Dependencies); j++ {
			options.CDependencies = appendUniquePath(options.CDependencies, header.Dependencies[j])
		}
		for j := 0; j < len(processed.Dependencies); j++ {
			options.CDependencies = appendUniquePath(options.CDependencies, processed.Dependencies[j])
			if executable {
				implementation := cLibcImplementation(processed.Dependencies[j])
				if implementation != "" && findString(selectedLibc, implementation) < 0 {
					libraryPath := load.JoinPath(libcRoot, "src/"+implementation)
					src, ok := fs.ReadFile(libraryPath)
					if !ok {
						return sourceFail(result, SourceErrReadFile, libraryPath)
					}
					selectedLibc = append(selectedLibc, implementation)
					result.Files = append(result.Files, load.SourceFile{Path: load.JoinPath(workDir, "__renvo_libc_"+implementation), Src: src})
					if implementation == "stdio.c" || implementation == "stdlib.c" || implementation == "assert.c" {
						needsRuntime = true
					}
				}
			}
		}
	}
	if executable && reloadGoFiles && len(pragmaGoFiles) > 0 {
		files := make([]string, len(options.Files), len(options.Files)+len(pragmaGoFiles))
		copy(files, options.Files)
		for i := 0; i < len(pragmaGoFiles); i++ {
			if findString(files, pragmaGoFiles[i]) < 0 {
				files = append(files, pragmaGoFiles[i])
			}
		}
		collected := CollectSourceFilesForTargetTagsWithModuleCache(
			workDir, stdRoot, files, options.Target, options.Tags, moduleCache, fs)
		if !collected.Ok {
			return collected
		}
		options.Files = files
		return prepareCSourcesPass(collected, options, workDir, stdRoot, moduleCache, fs, false)
	}
	if executable && firstC >= 0 && len(selectedLibc) > 0 {
		total := len(result.Files[firstC].Src)
		for i := 0; i < len(result.Files); i++ {
			if bundledCLibrarySource(result.Files[i].Path) {
				total += len(result.Files[i].Src) + 1
			}
		}
		merged := make([]byte, 0, total)
		merged = append(merged, result.Files[firstC].Src...)
		for i := 0; i < len(result.Files); i++ {
			if bundledCLibrarySource(result.Files[i].Path) {
				merged = append(merged, '\n')
				merged = append(merged, result.Files[i].Src...)
			}
		}
		result.Files[firstC].Src = merged
		files := result.Files[:0]
		for i := 0; i < len(result.Files); i++ {
			if bundledCLibrarySource(result.Files[i].Path) {
				continue
			}
			files = append(files, result.Files[i])
		}
		result.Files = files
	}
	if executable && needsRuntime {
		result.Files = append(result.Files, load.SourceFile{
			Path: load.JoinPath(workDir, "__renvo_c_runtime.go"),
			Src:  []byte("package main\nfunc __renvo_c_write_byte(fd int32, ch int32) int32 { data := []byte{byte(ch)}; if write(int(fd), data, -1) != 1 { return -1 }; return ch }\nfunc __renvo_c_read_byte(fd int32) int32 { data := []byte{0}; if read(int(fd), data, -1) != 1 { return -1 }; return int32(data[0]) }\nfunc renvo_runtime_Exit(status int32) {}\nfunc __renvo_c_abort(status int32) { renvo_runtime_Exit(status) }\n"),
		})
	}
	return result
}

func bundledCLibrarySource(path string) bool {
	name := load.BasePath(path)
	const prefix = "__renvo_libc_"
	return len(name) > len(prefix) && name[:len(prefix)] == prefix
}

func cLibcImplementation(path string) string {
	switch load.BasePath(path) {
	case "string.h":
		return "string.c"
	case "stdlib.h":
		return "stdlib.c"
	case "stdio.h":
		return "stdio.c"
	case "ctype.h":
		return "ctype.c"
	case "assert.h":
		return "assert.c"
	}
	return ""
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
			bestVersion := ""
			for j := 0; j < len(versions); j++ {
				if !versions[j].IsDir || !gccVersionNewer(versions[j].Name, bestVersion) {
					continue
				}
				include := load.JoinPath(load.JoinPath(targetRoot, versions[j].Name), "include")
				if fs.PathExists(include) {
					bestVersion = versions[j].Name
				}
			}
			if bestVersion != "" {
				paths = appendUniquePath(paths, load.JoinPath(
					load.JoinPath(targetRoot, bestVersion), "include"))
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

func gccVersionNewer(candidate string, current string) bool {
	if !validGCCVersion(candidate) {
		return false
	}
	if current == "" {
		return true
	}
	left := 0
	right := 0
	for left < len(candidate) || right < len(current) {
		leftPart, leftNext := gccVersionPart(candidate, left)
		rightPart, rightNext := gccVersionPart(current, right)
		leftPart = trimVersionZeros(leftPart)
		rightPart = trimVersionZeros(rightPart)
		if len(leftPart) != len(rightPart) {
			return len(leftPart) > len(rightPart)
		}
		if leftPart != rightPart {
			return leftPart > rightPart
		}
		left = leftNext
		right = rightNext
	}
	return candidate > current
}

func validGCCVersion(version string) bool {
	if len(version) == 0 || version[0] == '.' || version[len(version)-1] == '.' {
		return false
	}
	previousDot := false
	for i := 0; i < len(version); i++ {
		if version[i] == '.' {
			if previousDot {
				return false
			}
			previousDot = true
			continue
		}
		if version[i] < '0' || version[i] > '9' {
			return false
		}
		previousDot = false
	}
	return true
}

func gccVersionPart(version string, at int) (string, int) {
	if at >= len(version) {
		return "", at
	}
	end := at
	for end < len(version) && version[end] != '.' {
		end++
	}
	next := end
	if next < len(version) {
		next++
	}
	return version[at:end], next
}

func trimVersionZeros(part string) string {
	for len(part) > 0 && part[0] == '0' {
		part = part[1:]
	}
	return part
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
