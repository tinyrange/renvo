package main

import (
	"renvo.dev/ide"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
)

const editorAnalysisTimerID = 73

type editorAnalysisSession struct {
	path    string
	target  string
	source  []byte
	files   []load.SourceFile
	result  driver.AnalysisResult
	version int
	ready   bool
}

func (f *MainForm) editorChanged() {
	f.requestEditorAnalysis()
}

func (f *MainForm) requestEditorAnalysis() {
	if f == nil || f.editor == nil || f.editor.Document == nil || f.currentPath == "" {
		return
	}
	f.analysis.version++
	f.analysisTimer = true
}

func (f *MainForm) takeEditorAnalysisTimer() bool {
	if f == nil || !f.analysisTimer {
		return false
	}
	f.analysisTimer = false
	return true
}

func (f *MainForm) runEditorAnalysis() {
	if f == nil || f.editor == nil || f.editor.Document == nil {
		return
	}
	version := f.analysis.version
	source := []byte(f.editor.Document.Text())
	if !f.ensureEditorAnalysis(source) || version != f.analysis.version {
		return
	}
	diagnostic := f.analysis.result.Diagnostic
	if !diagnostic.Valid() {
		f.editor.SetDiagnostics(nil)
		f.editorFrame.SetDiagnostic("")
		return
	}
	f.editorFrame.SetDiagnostic(diagnostic.Message)
	if load.CleanPath(diagnostic.Path) != load.CleanPath(f.currentPath) {
		f.editor.SetDiagnostics(nil)
		return
	}
	f.editor.SetDiagnostics([]ide.Diagnostic{{Start: diagnostic.Start, End: diagnostic.End, Message: diagnostic.Message, Error: true}})
}

func (f *MainForm) ensureEditorAnalysis(source []byte) bool {
	if f == nil || f.currentPath == "" || f.root == "" {
		return false
	}
	path := load.CleanPath(f.currentPath)
	if f.analysis.ready && f.analysis.path == path && f.analysis.target == f.selectedTarget && analysisBytesEqual(f.analysis.source, source) {
		return true
	}
	files := analysisOverlayFiles(f.analysis.files, path, source)
	if len(files) == 0 || f.analysis.target != f.selectedTarget {
		files = f.collectEditorAnalysisFiles(path, source)
	}
	if len(files) == 0 {
		return false
	}
	result := driver.AnalyzeWorkspace(f.root, completionStdRoot(f.env), ".", files)
	if !result.Workspace.Ok && len(f.analysis.files) > 0 {
		refreshed := f.collectEditorAnalysisFiles(path, source)
		if len(refreshed) > 0 {
			files = refreshed
			result = driver.AnalyzeWorkspace(f.root, completionStdRoot(f.env), ".", files)
		}
	}
	f.analysis.path = path
	f.analysis.target = f.selectedTarget
	analysisReplaceSource(&f.analysis, source)
	f.analysis.files = files
	f.analysis.result = result
	f.analysis.ready = true
	return true
}

func analysisReplaceSource(session *editorAnalysisSession, source []byte) {
	copyOfSource := make([]byte, len(source))
	copy(copyOfSource, source)
	session.source = copyOfSource
}

func (f *MainForm) collectEditorAnalysisFiles(path string, source []byte) []load.SourceFile {
	fs := completionOverlayFS{base: completionSourceFS(), path: path, data: source}
	sources := driver.CollectSourcesForTargetTagsWithModuleCache(f.root, completionStdRoot(f.env), ".", f.selectedTarget, nil, completionModuleCache(f.env), fs)
	if !sources.Ok && sources.Error != driver.SourceErrParse {
		return nil
	}
	return sources.Files
}

func analysisOverlayFiles(files []load.SourceFile, path string, source []byte) []load.SourceFile {
	if len(files) == 0 {
		return nil
	}
	out := make([]load.SourceFile, len(files))
	copy(out, files)
	found := false
	for i := 0; i < len(out); i++ {
		if load.CleanPath(out[i].Path) == path {
			out[i].Src = source
			found = true
		}
	}
	if !found {
		return nil
	}
	return out
}

func analysisBytesEqual(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for i := 0; i < len(left); i++ {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
