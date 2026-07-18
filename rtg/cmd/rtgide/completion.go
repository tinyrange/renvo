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
	if !f.ensureEditorAnalysis(source) || !f.analysis.result.Workspace.Ok {
		return nil
	}
	path := load.CleanPath(f.currentPath)
	var semantic []check.CompletionItem
	if f.analysis.result.Program.Ok {
		semantic = check.CompleteProgram(f.analysis.result.Workspace.Graph, f.analysis.result.Program, path, caret)
	} else {
		semantic = check.CompleteGraph(f.analysis.result.Workspace.Graph, path, caret)
	}
	items := make([]ide.Completion, 0, len(semantic))
	for i := 0; i < len(semantic); i++ {
		parameters := make([]ide.SignatureParameter, 0, len(semantic[i].Parameters))
		for j := 0; j < len(semantic[i].Parameters); j++ {
			parameters = append(parameters, ide.SignatureParameter{Name: semantic[i].Parameters[j].Name, Type: semantic[i].Parameters[j].Type})
		}
		items = append(items, ide.Completion{Text: semantic[i].Name, Detail: semantic[i].Detail, Kind: semantic[i].Kind, Signature: semantic[i].Signature, Parameters: parameters})
	}
	return items
}

func (f *MainForm) signatureEditor(source []byte, caret int, out *ide.SignatureHelp) {
	if !f.ensureEditorAnalysis(source) || !f.analysis.result.Workspace.Ok {
		return
	}
	var help check.SignatureHelp
	if f.analysis.result.Program.Ok {
		help = check.SignatureHelpProgram(f.analysis.result.Workspace.Graph, f.analysis.result.Program, load.CleanPath(f.currentPath), caret)
	} else {
		help = check.SignatureHelpGraph(f.analysis.result.Workspace.Graph, load.CleanPath(f.currentPath), caret)
	}
	parameters := make([]ide.SignatureParameter, 0, len(help.Parameters))
	for i := 0; i < len(help.Parameters); i++ {
		parameters = append(parameters, ide.SignatureParameter{Name: help.Parameters[i].Name, Type: help.Parameters[i].Type})
	}
	*out = ide.SignatureHelp{Ok: help.Ok, Label: help.Label, Parameters: parameters, ActiveParameter: help.ActiveParameter}
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
