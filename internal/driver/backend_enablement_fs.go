//go:build !renvo

package driver

import (
	"renvo.dev/internal/load"
	"renvo.dev/internal/rbe"
)

// backendEnablementFS overlays files carried by an RBE (or prepared RTGB) on
// the configured standard-library root. The user's project filesystem remains
// otherwise untouched.
type backendEnablementFS struct {
	base    SourceFS
	stdRoot string
	files   []rbe.File
}

func (fs backendEnablementFS) ReadFile(path string) ([]byte, bool) {
	path = load.CleanPath(path)
	for i := range fs.files {
		if path == load.JoinPath(fs.stdRoot, fs.files[i].Path) {
			return append([]byte(nil), fs.files[i].Source...), true
		}
	}
	return fs.base.ReadFile(path)
}

func (fs backendEnablementFS) ReadDir(path string) ([]DirEntry, bool) {
	path = load.CleanPath(path)
	entries, baseOK := fs.base.ReadDir(path)
	out := append([]DirEntry(nil), entries...)
	found := false
	for i := range fs.files {
		full := load.JoinPath(fs.stdRoot, fs.files[i].Path)
		relative, within := load.RelPath(path, full)
		if !within || relative == "" || relative == "." {
			continue
		}
		name := relative
		isDir := false
		for j := 0; j < len(relative); j++ {
			if relative[j] == '/' {
				name = relative[:j]
				isDir = true
				break
			}
		}
		if name == "" {
			continue
		}
		found = true
		index := -1
		for j := range out {
			if out[j].Name == name {
				index = j
				break
			}
		}
		if index < 0 {
			out = append(out, DirEntry{Name: name, IsDir: isDir})
		} else if isDir {
			out[index].IsDir = true
		}
	}
	if !baseOK && !found {
		return nil, false
	}
	sortDirEntries(out)
	return out, true
}

func (fs backendEnablementFS) PathExists(path string) bool {
	path = load.CleanPath(path)
	if fs.base.PathExists(path) {
		return true
	}
	for i := range fs.files {
		full := load.JoinPath(fs.stdRoot, fs.files[i].Path)
		if path == full {
			return true
		}
		if relative, within := load.RelPath(path, full); within && relative != "" && relative != "." {
			return true
		}
	}
	return false
}
