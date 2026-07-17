package main

import (
	"j5.nz/rtg/rtg/forms"
	"j5.nz/rtg/rtg/ide"
	"j5.nz/rtg/rtg/std/graphics"
	rtgos "j5.nz/rtg/rtg/std/os"
)

// MainForm contains one IDE window. main_form_generated.go owns construction
// and property assignment; this file contains application state and callbacks.
type MainForm struct {
	forms.Form
	appBar        *workspaceAppBar
	explorerFrame *workspaceExplorerFrame
	editorFrame   *workspaceEditorFrame
	designer      *workspaceDesigner
	inspector     *workspaceInspector
	output        *workspaceOutput
	explorer      *ide.ExplorerControl
	editor        *ide.EditorControl
	currentPath   string
	root          string
	env           []string
	designerView  bool
	lastBuildOK   bool
	projectOutput string
}

func NewMainForm(root string) *MainForm {
	return NewMainFormWithEnv(root, nil)
}

func NewMainFormWithEnv(root string, env []string) *MainForm {
	form := &MainForm{root: root, projectOutput: workspaceJoinPath(root, projectOutputFile)}
	form.env = append(form.env, env...)
	created, message := ensureHelloWorldProject(root)
	form.initializeComponent(root)
	if message != "" {
		form.output.SetMessage(message, created)
	}
	initial := workspaceJoinPath(root, projectUserFormFile)
	if data, err := rtgos.ReadFile(initial); err == nil {
		form.currentPath = initial
		form.editor.SetDocument(ide.NewDocument(data))
		form.syncEditorFrame()
	}
	return form
}

func (f *MainForm) explorerOpenFile(path string) {
	data, err := rtgos.ReadFile(path)
	if err != nil {
		return
	}
	f.currentPath = path
	f.editor.SetDocument(ide.NewDocument(data))
	f.syncEditorFrame()
	f.showCode()
}

func (f *MainForm) saveCurrentFile() {
	if f.currentPath == "" || f.editor.Document == nil || !f.editor.Document.Dirty() {
		return
	}
	if rtgos.WriteFile(f.currentPath, f.editor.Document.Bytes(), 0644) == nil {
		f.editor.Document.MarkSaved()
		f.editor.Invalidate()
		f.syncEditorFrame()
	}
}

func (f *MainForm) formResize() {
	width, height := f.Size()
	layout := calculateWorkspaceLayout(width, height)
	f.appBar.SetBounds(rect(0, 0, width, workspaceAppBarHeight))
	f.explorerFrame.SetBounds(layout.explorerFrame)
	f.editorFrame.SetBounds(layout.editorFrame)
	f.designer.SetBounds(layout.designer)
	f.inspector.SetBounds(layout.inspector)
	f.output.SetBounds(layout.output)
	f.explorer.SetBounds(layout.explorer)
	f.editor.SetBounds(layout.editor)
	f.syncEditorFrame()
}

func (f *MainForm) showCode() {
	if f == nil {
		return
	}
	f.designerView = false
	f.designer.SetVisible(false)
	f.editorFrame.SetVisible(true)
	f.editor.SetVisible(true)
}

func (f *MainForm) showDesigner() {
	if f == nil {
		return
	}
	f.designerView = true
	f.editorFrame.SetVisible(false)
	f.editor.SetVisible(false)
	f.designer.SetVisible(true)
}

func (f *MainForm) buildProject() {
	f.saveCurrentFile()
	f.output.SetMessage("Building the project…", true)
	result := compileIDEProject(f.root, f.projectOutput, f.env)
	f.lastBuildOK = result.ok
	f.output.SetMessage(result.message, result.ok)
}

func (f *MainForm) runProject() {
	if !f.lastBuildOK || (f.editor.Document != nil && f.editor.Document.Dirty()) {
		f.buildProject()
	}
	if !f.lastBuildOK {
		return
	}
	result := launchIDEProject(f.projectOutput, f.root)
	f.output.SetMessage(result.message, result.ok)
}

// Dispatch keeps the working editor model and the surrounding status chrome
// synchronized without coupling editor commands to this particular shell.
func (f *MainForm) Dispatch(event graphics.Event) {
	f.Form.Dispatch(event)
	if f.editor.Document != nil && f.editor.Document.Dirty() {
		f.lastBuildOK = false
	}
	f.syncEditorFrame()
}

func (f *MainForm) syncEditorFrame() {
	if f.editorFrame == nil || f.editor == nil || f.editor.Document == nil {
		return
	}
	line, column := f.editor.Document.Position(f.editor.Document.Caret)
	f.editorFrame.SetDocumentState(f.currentPath, f.editor.Document.Dirty(), line+1, column+1)
}
