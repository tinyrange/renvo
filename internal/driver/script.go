package driver

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/syntax"
)

// scriptSource turns a deliberately small script surface into an ordinary
// main package. Imports remain package declarations; every other token is
// placed in main. The generated prefix contains no newline so source line
// numbers remain stable in diagnostics.
func scriptSource(src []byte) []byte {
	importEnd := scriptImportPrefixEnd(src)
	out := make([]byte, 0, len(src)+40)
	out = append(out, "package main;"...)
	out = append(out, src[:importEnd]...)
	out = append(out, "\nfunc main(){"...)
	out = append(out, src[importEnd:]...)
	out = append(out, "\n}\n"...)
	return out
}

func scriptImportPrefixEnd(src []byte) int {
	tokens := syntax.Scan(src)
	pos := 0
	last := 0
	for pos < len(tokens) {
		if tokens[pos].KindLine&255 != syntax.TokenImport {
			return last
		}
		pos++
		if pos < len(tokens) && tokens[pos].KindLine>>syntax.TokenOperatorCharShift&syntax.TokenOperatorCharMask == int('(') {
			depth := 0
			for pos < len(tokens) {
				ch := tokens[pos].KindLine >> syntax.TokenOperatorCharShift & syntax.TokenOperatorCharMask
				if ch == int('(') {
					depth++
				} else if ch == int(')') {
					depth--
				}
				last = syntax.TokenEnd(tokens[pos])
				pos++
				if depth == 0 {
					break
				}
			}
			if depth != 0 {
				return len(src)
			}
		} else {
			found := false
			for pos < len(tokens) {
				if tokens[pos].KindLine&255 == syntax.TokenString {
					last = syntax.TokenEnd(tokens[pos])
					pos++
					found = true
					break
				}
				if tokens[pos].KindLine&255 == syntax.TokenEOF {
					break
				}
				pos++
			}
			if !found {
				return len(src)
			}
		}
		if pos < len(tokens) && tokens[pos].KindLine>>syntax.TokenOperatorCharShift&syntax.TokenOperatorCharMask == int(';') {
			last = syntax.TokenEnd(tokens[pos])
			pos++
		}
	}
	return last
}

func transformScriptFiles(files []load.SourceFile, workDir string, options Options) []load.SourceFile {
	if !options.Script || len(options.Files) != 1 {
		return files
	}
	scriptPath := load.CleanPath(load.JoinPath(workDir, options.Files[0]))
	out := make([]load.SourceFile, len(files))
	copy(out, files)
	for i := 0; i < len(out); i++ {
		if load.CleanPath(out[i].Path) == scriptPath {
			out[i].Src = scriptSource(out[i].Src)
			out[i].ArenaStart = 0
			out[i].ArenaEnd = 0
		}
	}
	return out
}

type scriptSourceFS struct {
	base SourceFS
	path string
}

type standaloneCSourceFS struct {
	base       SourceFS
	modulePath string
	synthetic  bool
}

func (fs standaloneCSourceFS) ReadDir(path string) ([]DirEntry, bool) {
	entries, ok := fs.base.ReadDir(path)
	return entries, ok
}

func (fs standaloneCSourceFS) ReadFile(path string) ([]byte, bool) {
	if fs.synthetic && load.CleanPath(path) == fs.modulePath {
		return []byte("module renvo.c.program\n"), true
	}
	src, ok := fs.base.ReadFile(path)
	return src, ok
}

func (fs standaloneCSourceFS) PathExists(path string) bool {
	if fs.synthetic && load.CleanPath(path) == fs.modulePath {
		return true
	}
	return fs.base.PathExists(path)
}

func standaloneCSourceContext(workDir string, options Options) (string, Options) {
	if len(options.Files) == 0 {
		return workDir, options
	}
	// Object mode has historically accepted a single source outside the current
	// module, including Go sources. Preserve that behavior while extending the
	// same standalone context to multi-file C compilations below.
	if options.Mode == ModeObject && len(options.Files) == 1 {
		path := load.CleanPath(load.JoinPath(workDir, options.Files[0]))
		workDir = load.DirPath(path)
		name := load.BasePath(path)
		options.Package = name
		options.Files = []string{name}
		return workDir, options
	}
	dir := ""
	files := make([]string, len(options.Files))
	for i := 0; i < len(options.Files); i++ {
		path := load.CleanPath(load.JoinPath(workDir, options.Files[i]))
		if !isFrontendSourceName(load.BasePath(path)) {
			return workDir, options
		}
		if dir == "" {
			dir = load.DirPath(path)
		} else if dir != load.DirPath(path) {
			return workDir, options
		}
		files[i] = load.BasePath(path)
	}
	workDir = dir
	options.Package = files[0]
	options.Files = files
	return workDir, options
}

// Kbuild invokes the compiler from the kernel root while naming sources in
// subdirectories. Resolve search and forced-include paths before the ordinary
// single-file object path changes its loader context to the source directory.
func resolveCCompilerPaths(workDir string, options Options) Options {
	if !options.CCompiler {
		return options
	}
	options.CDependencyRoot = load.CleanPath(workDir)
	for i := 0; i < len(options.IncludePaths); i++ {
		options.IncludePaths[i] = load.JoinPath(workDir, options.IncludePaths[i])
	}
	for i := 0; i < len(options.CForcedInclude); i++ {
		options.CForcedInclude[i] = load.JoinPath(workDir, options.CForcedInclude[i])
	}
	return options
}

func (fs scriptSourceFS) ReadDir(path string) ([]DirEntry, bool) {
	entries, ok := fs.base.ReadDir(path)
	return entries, ok
}

func (fs scriptSourceFS) ReadFile(path string) ([]byte, bool) {
	src, ok := fs.base.ReadFile(path)
	if ok && load.CleanPath(path) == fs.path {
		src = scriptSource(src)
	}
	return src, ok
}

func (fs scriptSourceFS) PathExists(path string) bool {
	return fs.base.PathExists(path)
}

func sourceFSForOptions(fs SourceFS, workDir string, options Options) SourceFS {
	if len(options.Files) > 0 {
		_, _, _, hasModule := findModuleSource(workDir, fs)
		fs = standaloneCSourceFS{
			base:       fs,
			modulePath: load.CleanPath(load.JoinPath(workDir, "go.mod")),
			synthetic:  !hasModule,
		}
	}
	if options.Script && len(options.Files) == 1 {
		fs = scriptSourceFS{
			base: fs,
			path: load.CleanPath(load.JoinPath(workDir, options.Files[0])),
		}
	}
	return fs
}
