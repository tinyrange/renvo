package main

import (
	"j5.nz/rtg/rtg/ide"
	"j5.nz/rtg/rtg/internal/check"
	"j5.nz/rtg/rtg/internal/driver"
	"j5.nz/rtg/rtg/internal/load"
)

type completionOverlayFS struct {
	base driver.SourceFS
	path string
	data []byte
}

func (fs completionOverlayFS) ReadDir(path string) ([]driver.DirEntry, bool) {
	return fs.base.ReadDir(path)
}

func (fs completionOverlayFS) ReadFile(path string) ([]byte, bool) {
	if load.CleanPath(path) == fs.path {
		return fs.data, true
	}
	return fs.base.ReadFile(path)
}

func (f *MainForm) completeEditor(source []byte, caret int) []ide.Completion {
	if f.currentPath == "" || f.root == "" {
		return nil
	}
	path := load.CleanPath(f.currentPath)
	fs := completionOverlayFS{base: completionSourceFS(), path: path, data: source}
	sources := driver.CollectSourcesForTargetTagsWithModuleCache(f.root, completionStdRoot(f.env), ".", f.selectedTarget, nil, completionModuleCache(f.env), fs)
	if !sources.Ok {
		return nil
	}
	workspace := load.LoadWorkspace(f.root, completionStdRoot(f.env), ".", sources.Files)
	if !workspace.Ok {
		return nil
	}
	semantic := check.CompleteGraph(workspace.Graph, path, caret)
	items := make([]ide.Completion, 0, len(semantic))
	for i := 0; i < len(semantic); i++ {
		items = append(items, ide.Completion{Text: semantic[i].Name, Detail: semantic[i].Detail, Kind: semantic[i].Kind})
	}
	return items
}

func completionEnv(env []string, key string) string {
	prefix := key + "="
	for i := 0; i < len(env); i++ {
		if workspaceHasPrefix(env[i], prefix) {
			return env[i][len(prefix):]
		}
	}
	return ""
}
