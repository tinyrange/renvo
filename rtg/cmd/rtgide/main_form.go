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
	appBar         *workspaceAppBar
	targetMenu     *workspaceTargetMenu
	explorerFrame  *workspaceExplorerFrame
	editorFrame    *workspaceEditorFrame
	designer       *workspaceDesigner
	inspector      *workspaceInspector
	output         *workspaceOutput
	explorer       *ide.ExplorerControl
	editor         *ide.EditorControl
	currentPath    string
	root           string
	env            []string
	design         formDesign
	designerView   bool
	lastBuildOK    bool
	projectOutput  string
	selectedTarget string
	analysis       editorAnalysisSession
	analysisTimer  bool
}

func NewMainForm(root string) *MainForm {
	return NewMainFormWithEnv(root, nil)
}

func NewMainFormWithEnv(root string, env []string) *MainForm {
	target := defaultIDETarget()
	if target == "" {
		target = workspaceTargets()[0]
	}
	form := &MainForm{root: root, selectedTarget: target, projectOutput: workspaceProjectOutput(root, target), design: defaultFormDesign()}
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
		form.requestEditorAnalysis()
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
	f.requestEditorAnalysis()
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
	if !f.designerView {
		documentX := int(layout.editorFrame.MinX)
		layout.editorFrame = rect(documentX, workspaceAppBarHeight, width-documentX, int(layout.editorFrame.Height()))
		layout.output = rect(documentX, int(layout.output.MinY), width-documentX, int(layout.output.Height()))
		layout.editor = rect(documentX, int(layout.editor.MinY), width-documentX, int(layout.editor.Height()))
	}
	f.appBar.SetBounds(rect(0, 0, width, workspaceAppBarHeight))
	f.targetMenu.SetBounds(rect(170, 39, 184, len(workspaceTargets())*27+10))
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
	f.inspector.SetVisible(false)
	f.editorFrame.SetVisible(true)
	f.editor.SetVisible(true)
	f.formResize()
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
	f.inspector.SetVisible(true)
	f.formResize()
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
		height = 36
	} else if kind == designerTextBox {
		base = "textBox"
		text = "Text input"
		width = 180
		height = 32
	} else if kind == designerTextArea {
		base = "textArea"
		text = "Text area"
		width = 220
		height = 90
	} else if kind == designerCheckBox {
		base = "checkBox"
		text = "Check box"
		width = 140
	} else if kind == designerRadioButton {
		base = "radioButton"
		text = "Radio button"
		width = 150
	} else if kind == designerPictureBox {
		base = "pictureBox"
		text = ""
		width = 140
		height = 100
	} else if kind == designerPanel {
		base = "panel"
		text = ""
		width = 240
		height = 140
	}
	name := f.nextDesignerName(base)
	offset := len(f.design.controls) * designerGridSize
	x := designerSnap(20 + offset)
	y := designerSnap(20 + offset)
	if x+width > f.design.width {
		x = 20
	}
	if y+height > f.design.height {
		y = 20
	}
	control := designerControl{kind: kind, name: name, text: text, x: x, y: y, width: width, height: height}
	index := len(f.design.controls)
	if kind == designerPanel {
		f.design.controls = append(f.design.controls, designerControl{})
		copy(f.design.controls[1:], f.design.controls[:len(f.design.controls)-1])
		f.design.controls[0] = control
		index = 0
	} else {
		f.design.controls = append(f.design.controls, control)
	}
	f.designer.SetSelection(index)
	f.inspector.SetSelection(index)
	f.designer.Invalidate()
	f.inspector.Invalidate()
	f.persistDesigner()
}

func (f *MainForm) deleteDesignerControl(index int) {
	if index < 0 || index >= len(f.design.controls) {
		return
	}
	copy(f.design.controls[index:], f.design.controls[index+1:])
	f.design.controls = f.design.controls[:len(f.design.controls)-1]
	next := index
	if next >= len(f.design.controls) {
		next = len(f.design.controls) - 1
	}
	f.designer.selected = -1
	f.inspector.selected = -1
	f.designer.SetSelection(next)
	f.inspector.SetSelection(next)
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

func (f *MainForm) createDesignerEvent(handler string, paint bool) {
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
	if paint {
		signature = "func (f *MainForm) " + handler + "(surface *graphics.Surface)"
	}
	if workspaceContains(string(data), signature) {
		return
	}
	if paint {
		data = ensureUserGraphicsImport(data)
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

func ensureUserGraphicsImport(data []byte) []byte {
	const importPath = `"j5.nz/rtg/rtg/std/graphics"`
	if workspaceContains(string(data), importPath) {
		return data
	}
	const packageLine = "package main\n"
	text := string(data)
	for i := 0; i+len(packageLine) <= len(text); i++ {
		if text[i:i+len(packageLine)] != packageLine {
			continue
		}
		out := make([]byte, 0, len(data)+len(importPath)+10)
		out = append(out, data[:i+len(packageLine)]...)
		out = append(out, '\n')
		out = append(out, "import "...)
		out = append(out, importPath...)
		out = append(out, "\n"...)
		out = append(out, data[i+len(packageLine):]...)
		return out
	}
	return data
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
	result := compileIDEProject(f.root, f.projectOutput, f.selectedTarget, f.env)
	f.lastBuildOK = result.ok
	f.output.SetMessage(result.message, result.ok)
}

func (f *MainForm) runProject() {
	// Always rebuild before Run. Files can be edited by another process, and a
	// cached successful result does not prove that the executable still matches
	// the project currently on disk.
	f.buildProject()
	if !f.lastBuildOK {
		return
	}
	if f.selectedTarget != defaultIDETarget() {
		f.output.SetMessage("Build succeeded for "+f.selectedTarget+". Run is available only for "+defaultIDETarget()+" on this host.", true)
		return
	}
	result := launchIDEProject(f.projectOutput, f.root)
	f.output.SetMessage(result.message, result.ok)
}

func (f *MainForm) toggleTargetMenu() {
	f.targetMenu.SetVisible(!f.targetMenu.Visible())
}

func (f *MainForm) selectBuildTarget(target string) {
	if target == "" {
		return
	}
	f.targetMenu.SetVisible(false)
	if target == f.selectedTarget {
		return
	}
	f.selectedTarget = target
	f.projectOutput = workspaceProjectOutput(f.root, target)
	f.lastBuildOK = false
	f.appBar.SetTarget(target)
	f.output.SetMessage("Build target: "+target, true)
	f.analysis = editorAnalysisSession{}
	f.requestEditorAnalysis()
}

func workspaceTargets() []string {
	return []string{"darwin/arm64", "windows/amd64", "windows/386", "windows/arm64"}
}

func workspaceProjectOutput(root, target string) string {
	name := projectOutputFile
	if workspaceHasPrefix(target, "windows/") {
		name += ".exe"
	} else if target == "wasi/wasm32" {
		name += ".wasm"
	}
	return workspaceJoinPath(root, name)
}

// Dispatch keeps the working editor model and the surrounding status chrome
// synchronized without coupling editor commands to this particular shell.
func (f *MainForm) Dispatch(event graphics.Event) {
	if event.Type == graphics.EventTimer && event.TimerID == editorAnalysisTimerID {
		f.runEditorAnalysis()
	}
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
