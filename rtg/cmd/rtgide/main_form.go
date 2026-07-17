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
	design        formDesign
	designerView  bool
	lastBuildOK   bool
	projectOutput string
}

func NewMainForm(root string) *MainForm {
	return NewMainFormWithEnv(root, nil)
}

func NewMainFormWithEnv(root string, env []string) *MainForm {
	form := &MainForm{root: root, projectOutput: workspaceJoinPath(root, projectOutputFile), design: defaultFormDesign()}
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
	f.saveCurrentFile()
	data, err := rtgos.ReadFile(workspaceJoinPath(f.root, projectGeneratedFormFile))
	if err != nil {
		f.output.SetMessage("Could not read "+projectGeneratedFormFile+".", false)
		return
	}
	design, message := parseFormDesign(data)
	if message != "" {
		f.output.SetMessage(message, false)
		return
	}
	f.design = design
	f.designer.SetDesign(&f.design)
	f.inspector.SetDesign(&f.design)
	f.designerView = true
	f.editorFrame.SetVisible(false)
	f.editor.SetVisible(false)
	f.designer.SetVisible(true)
}

func (f *MainForm) designerSelectionChanged(index int) {
	f.inspector.SetSelection(index)
}

func (f *MainForm) addDesignerControl(kind string) {
	base := "label"
	text := "Label"
	width := 120
	height := 28
	if kind == designerButton {
		base = "button"
		text = "Button"
		width = 120
		height = 36
	}
	name := f.nextDesignerName(base)
	offset := len(f.design.controls) * 12
	x := 24 + offset
	y := 24 + offset
	if x+width > f.design.width {
		x = 24
	}
	if y+height > f.design.height {
		y = 24
	}
	f.design.controls = append(f.design.controls, designerControl{kind: kind, name: name, text: text, x: x, y: y, width: width, height: height})
	index := len(f.design.controls) - 1
	f.designer.SetSelection(index)
	f.inspector.SetSelection(index)
	f.designer.Invalidate()
	f.inspector.Invalidate()
	f.persistDesigner()
}

func (f *MainForm) nextDesignerName(base string) string {
	for number := 1; ; number++ {
		candidate := base + workspaceDecimal(number)
		used := false
		for i := 0; i < len(f.design.controls); i++ {
			if f.design.controls[i].name == candidate {
				used = true
				break
			}
		}
		if !used {
			return candidate
		}
	}
}

func (f *MainForm) designerChanged() {
	f.designer.InvalidatePreview()
	f.inspector.InvalidateProperties()
	f.persistDesigner()
}

func (f *MainForm) persistDesigner() {
	path := workspaceJoinPath(f.root, projectGeneratedFormFile)
	data := generatedFormSource(f.design)
	if rtgos.WriteFile(path, data, 0644) != nil {
		f.output.SetMessage("Could not update "+projectGeneratedFormFile+".", false)
		return
	}
	f.lastBuildOK = false
	f.output.SetMessage("Designer changes saved to "+projectGeneratedFormFile+".", true)
	if f.explorer != nil && f.explorer.Model != nil {
		f.explorer.Model.Refresh()
		f.explorer.Invalidate()
	}
	if f.currentPath == path {
		f.editor.SetDocument(ide.NewDocument(data))
		f.syncEditorFrame()
	}
}

func (f *MainForm) createDesignerEvent(handler string) {
	if handler == "" {
		return
	}
	path := workspaceJoinPath(f.root, projectUserFormFile)
	data, err := rtgos.ReadFile(path)
	if err != nil {
		f.output.SetMessage("Could not read "+projectUserFormFile+".", false)
		return
	}
	signature := "func (f *MainForm) " + handler + "()"
	if workspaceContains(string(data), signature) {
		return
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	data = append(data, '\n')
	data = append(data, signature...)
	data = append(data, " {\n}\n"...)
	if rtgos.WriteFile(path, data, 0644) != nil {
		f.output.SetMessage("Could not create event handler in "+projectUserFormFile+".", false)
		return
	}
	if f.currentPath == path && (f.editor.Document == nil || !f.editor.Document.Dirty()) {
		f.editor.SetDocument(ide.NewDocument(data))
		f.syncEditorFrame()
	}
}

func workspaceContains(value, fragment string) bool {
	if fragment == "" {
		return true
	}
	for i := 0; i+len(fragment) <= len(value); i++ {
		if value[i:i+len(fragment)] == fragment {
			return true
		}
	}
	return false
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
