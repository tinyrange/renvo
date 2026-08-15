package driver

import (
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

func prepareCObjectSources(result SourceResult, options Options, workDir string, fs SourceFS) SourceResult {
	if !result.Ok || options.Mode != ModeObject {
		return result
	}
	reader := cObjectIncludeReader{fs: fs, paths: cObjectIncludePaths(workDir, options.IncludePaths, fs)}
	for i := 0; i < len(result.Files); i++ {
		if !optionArgIsCFile(result.Files[i].Path) {
			continue
		}
		header := c11.BuildObjectPrelude(result.Files[i].Path, result.Files[i].Src, reader)
		if !header.Ok {
			result = sourceFail(result, SourceErrCInclude, header.ErrorPath)
			result.ErrorSourcePath = result.Files[i].Path
			result.ErrorOffset = header.ErrorAt
			return result
		}
		result.Files[i].CObject = true
		result.Files[i].CPrelude = header.Prelude
	}
	return result
}

func cObjectIncludePaths(workDir string, explicit []string, fs SourceFS) []string {
	paths := make([]string, 0, len(explicit)+8)
	for i := 0; i < len(explicit); i++ {
		path := load.JoinPath(workDir, explicit[i])
		paths = appendUniquePath(paths, path)
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
	for _, path := range []string{"/usr/local/include", "/usr/include/x86_64-linux-gnu", "/usr/include"} {
		if fs.PathExists(path) {
			paths = appendUniquePath(paths, path)
		}
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
