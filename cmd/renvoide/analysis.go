package main

import (
	"renvo.dev/ide"
	"renvo.dev/internal/arena"
	"renvo.dev/internal/driver"
	"renvo.dev/internal/load"
)

const editorAnalysisTimerID = 73

type editorAnalysisSession struct {
	version int
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
	mark := arena.Mark()
	result, ok := f.analyzeEditorSource(source)
	if !ok {
		arena.Reset(mark)
		return
	}
	diagnostic := persistEditorDiagnostic(result.Diagnostic)
	arena.Reset(mark)
	if version != f.analysis.version {
		return
	}
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

func (f *MainForm) analyzeEditorSource(source []byte) (driver.AnalysisResult, bool) {
	if f == nil || f.currentPath == "" || f.root == "" {
		return driver.AnalysisResult{}, false
	}
	path := load.CleanPath(f.currentPath)
	files := f.collectEditorAnalysisFiles(path, source)
	if len(files) == 0 {
		return driver.AnalysisResult{}, false
	}
	result := driver.AnalyzeWorkspace(f.root, completionStdRoot(f.env), ".", files)
	return result, true
}

func persistEditorDiagnostic(diagnostic driver.Diagnostic) driver.Diagnostic {
	diagnostic.Phase = arena.PersistString(diagnostic.Phase)
	diagnostic.Code = arena.PersistString(diagnostic.Code)
	diagnostic.Message = arena.PersistString(diagnostic.Message)
	diagnostic.Path = arena.PersistString(diagnostic.Path)
	return diagnostic
}

func (f *MainForm) collectEditorAnalysisFiles(path string, source []byte) []load.SourceFile {
	fs := completionOverlayFS{path: path, data: source}
	sources := driver.CollectSourcesForTargetTagsWithModuleCache(f.root, completionStdRoot(f.env), ".", f.selectedTarget, nil, completionModuleCache(f.env), fs)
	if !sources.Ok && sources.Error != driver.SourceErrParse {
		return nil
	}
	return sources.Files
}
