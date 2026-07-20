package main

import (
	"renvo.dev/forms"
	"renvo.dev/ide"
	"renvo.dev/std/graphics"
	renvoos "renvo.dev/std/os"
)

// MainForm contains one IDE window. main_form_generated.go owns construction
// and property assignment; this file contains application state and callbacks.
type MainForm struct {
	forms.Form
	window         *graphics.Window
	menuBar        *forms.MenuBar
	lightThemeItem *forms.MenuItem
	darkThemeItem  *forms.MenuItem
	statusBar      *forms.StatusBar
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
	darkTheme      bool
	lastBuildOK    bool
	projectOutput  string
	selectedTarget string
	analysis       editorAnalysisSession
	analysisTimer  bool
	buildSession   *ideBuildSession
	buildRunAfter  bool
}

const projectBuildTimerID = 74

func (f *MainForm) useLightTheme() { f.setDarkTheme(false) }
func (f *MainForm) useDarkTheme()  { f.setDarkTheme(true) }

func (f *MainForm) setDarkTheme(dark bool) {
	if f == nil {
		return
	}
	f.darkTheme = dark
	theme := forms.LightTheme()
	if dark {
		theme = forms.DarkTheme()
	}
	applyWorkspaceTheme(theme)
	f.Form.ApplyTheme(theme)
	if f.lightThemeItem != nil {
		f.lightThemeItem.SetChecked(!dark)
	}
	if f.darkThemeItem != nil {
		f.darkThemeItem.SetChecked(dark)
	}
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
	if data, err := renvoos.ReadFile(initial); err == nil {
		form.currentPath = initial
		form.editor.SetDocument(ide.NewDocument(data))
		form.syncEditorFrame()
		form.requestEditorAnalysis()
	}
	return form
}

func (f *MainForm) SetWindow(window *graphics.Window) {
	if f == nil {
		return
	}
	f.window = window
	f.updateWindowTitle()
}

func (f *MainForm) explorerOpenFile(path string) {
	data, err := renvoos.ReadFile(path)
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
	if f.editor.Document == nil || !f.editor.Document.Dirty() {
		return
	}
	if f.currentPath == "" {
		f.saveCurrentFileAs()
		return
	}
	f.saveDocumentTo(f.currentPath)
}

func (f *MainForm) saveDocumentTo(path string) bool {
	if f == nil || path == "" || f.editor == nil || f.editor.Document == nil {
		return false
	}
	if renvoos.WriteFile(path, f.editor.Document.Bytes(), 0644) == nil {
		f.currentPath = path
		f.editor.Document.MarkSaved()
		f.editor.Invalidate()
		f.syncEditorFrame()
		if f.explorer != nil && f.explorer.Model != nil {
			f.explorer.Model.Refresh()
			f.explorer.Invalidate()
		}
		return true
	}
	f.output.SetMessage("Could not save "+path+".", false)
	return false
}

func (f *MainForm) chooseOpenFile() {
	if f == nil || f.window == nil {
		return
	}
	path, ok := f.window.OpenFileDialog(graphics.FileDialogOptions{Title: "Open File", InitialDirectory: f.root})
	if ok {
		f.saveCurrentFile()
		f.explorerOpenFile(path)
	}
}

func (f *MainForm) saveCurrentFileAs() {
	if f == nil || f.window == nil || f.editor == nil || f.editor.Document == nil {
		return
	}
	name := workspacePathBase(f.currentPath)
	if name == "" {
		name = "main.go"
	}
	path, ok := f.window.SaveFileDialog(graphics.FileDialogOptions{Title: "Save File As", InitialDirectory: f.root, DefaultName: name})
	if ok {
		f.saveDocumentTo(path)
	}
}

func (f *MainForm) chooseNewProject() {
	if f == nil || f.window == nil {
		return
	}
	path, ok := f.window.SelectFolderDialog(graphics.FolderDialogOptions{Title: "Create Project", InitialDirectory: f.root})
	if ok {
		f.openProjectPath(path, true)
	}
}

func (f *MainForm) chooseOpenProject() {
	if f == nil || f.window == nil {
		return
	}
	path, ok := f.window.SelectFolderDialog(graphics.FolderDialogOptions{Title: "Open Project", InitialDirectory: f.root})
	if ok {
		f.openProjectPath(path, false)
	}
}

func (f *MainForm) openProjectPath(root string, create bool) bool {
	if f == nil || root == "" {
		return false
	}
	if _, err := renvoos.ReadDir(root); err != nil {
		f.output.SetMessage("Could not read the project directory.", false)
		return false
	}
	f.saveCurrentFile()
	created := false
	message := "Opened project " + workspacePathBase(root) + "."
	if create {
		created, message = ensureHelloWorldProject(root)
		if message != "" && !created {
			f.output.SetMessage(message, false)
			return false
		}
	}
	f.root = root
	f.projectOutput = workspaceProjectOutput(root, f.selectedTarget)
	f.lastBuildOK = false
	f.analysis = editorAnalysisSession{}
	f.analysisTimer = false
	f.design = defaultFormDesign()
	f.designer.SetDesign(&f.design)
	f.inspector.SetDesign(&f.design)
	f.explorer.SetModel(ide.OpenExplorer(root))
	f.currentPath = ""
	f.editor.SetDocument(ide.NewDocument(nil))
	initial := workspaceJoinPath(root, projectUserFormFile)
	if _, err := renvoos.ReadFile(initial); err != nil {
		initial = workspaceJoinPath(root, projectMainFile)
	}
	if data, err := renvoos.ReadFile(initial); err == nil {
		f.currentPath = initial
		f.editor.SetDocument(ide.NewDocument(data))
	}
	f.showCode()
	f.requestEditorAnalysis()
	f.updateWindowTitle()
	if message != "" {
		f.output.SetMessage(message, true)
	}
	return true
}

func (f *MainForm) updateWindowTitle() {
	if f == nil || f.window == nil {
		return
	}
	name := workspacePathBase(f.root)
	if name == "" {
		name = f.root
	}
	if name == "" || name == "." {
		f.window.SetTitle("RENVO")
	} else {
		f.window.SetTitle("RENVO — " + name)
	}
}

func (f *MainForm) formResize() {
	width, _ := f.Size()
	content := f.DockClientBounds()
	contentY := int(content.MinY)
	contentHeight := int(content.Height())
	layout := calculateWorkspaceLayout(width, contentHeight)
	layout.explorerFrame = workspaceOffsetY(layout.explorerFrame, contentY)
	layout.editorFrame = workspaceOffsetY(layout.editorFrame, contentY)
	layout.designer = workspaceOffsetY(layout.designer, contentY)
	layout.inspector = workspaceOffsetY(layout.inspector, contentY)
	layout.output = workspaceOffsetY(layout.output, contentY)
	layout.explorer = workspaceOffsetY(layout.explorer, contentY)
	layout.editor = workspaceOffsetY(layout.editor, contentY)
	if !f.designerView {
		documentX := int(layout.editorFrame.MinX)
		layout.editorFrame = rect(documentX, contentY+workspaceAppBarHeight, width-documentX, int(layout.editorFrame.Height()))
		layout.output = rect(documentX, int(layout.output.MinY), width-documentX, int(layout.output.Height()))
		layout.editor = rect(documentX, int(layout.editor.MinY), width-documentX, int(layout.editor.Height()))
	}
	f.appBar.SetBounds(rect(0, contentY, width, workspaceAppBarHeight))
	f.targetMenu.SetBounds(rect(workspaceTargetX, contentY+workspaceAppBarHeight-1, workspaceTargetWidth, len(workspaceTargets())*workspaceTargetMenuRowHeight+workspaceTargetMenuPadding*2))
	f.explorerFrame.SetBounds(layout.explorerFrame)
	f.editorFrame.SetBounds(layout.editorFrame)
	f.designer.SetBounds(layout.designer)
	f.inspector.SetBounds(layout.inspector)
	f.output.SetBounds(layout.output)
	f.explorer.SetBounds(layout.explorer)
	f.editor.SetBounds(layout.editor)
	f.syncEditorFrame()
}

func workspaceOffsetY(bounds graphics.Rect, offset int) graphics.Rect {
	bounds.MinY += graphics.Scalar(offset)
	bounds.MaxY += graphics.Scalar(offset)
	return bounds
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
	data, err := renvoos.ReadFile(workspaceJoinPath(f.root, projectGeneratedFormFile))
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
	} else if kind == designerComboBox {
		base, text, width, height = "comboBox", "Combo box", 180, 32
	} else if kind == designerListBox {
		base, text, width, height = "listBox", "List item", 180, 110
	} else if kind == designerListView {
		base, text, width, height = "listView", "List view", 260, 140
	} else if kind == designerTreeView {
		base, text, width, height = "treeView", "Tree item", 200, 160
	} else if kind == designerTabControl {
		base, text, width, height = "tabControl", "Tab page", 300, 44
	} else if kind == designerProgressBar {
		base, text, width, height = "progressBar", "", 220, 24
	} else if kind == designerNumericUpDown {
		base, text, width, height = "numericUpDown", "", 120, 32
	} else if kind == designerSlider {
		base, text, width, height = "slider", "", 220, 32
	} else if kind == designerGroupBox {
		base, text, width, height = "groupBox", "Group", 260, 150
	} else if kind == designerSplitContainer {
		base, text, width, height = "splitContainer", "", 320, 180
	} else if kind == designerToolBar {
		base, text, width, height = "toolBar", "Toolbar", 320, 36
	} else if kind == designerStatusBar {
		base, text, width, height = "statusBar", "Ready", 320, 28
	} else if kind == designerMenuBar {
		base, text, width, height = "menuBar", "", 320, 34
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
	value := 0
	if kind == designerProgressBar || kind == designerSlider {
		value = 50
	} else if kind == designerSplitContainer {
		value = width / 2
	}
	dock := ""
	if kind == designerMenuBar {
		dock = designerDockTop
		x = 0
		y = 0
		width = f.design.width
	} else if kind == designerStatusBar {
		dock = designerDockBottom
		x = 0
		y = f.design.height - height
		width = f.design.width
	}
	control := designerControl{kind: kind, name: name, text: text, dock: dock, x: x, y: y, width: width, height: height, value: value}
	index := len(f.design.controls)
	if kind == designerPanel || kind == designerGroupBox || kind == designerSplitContainer {
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
	f.designer.AccessibilityChildrenStateChanged()
	f.inspector.InvalidateProperties()
	f.persistDesigner()
}

func (f *MainForm) persistDesigner() {
	path := workspaceJoinPath(f.root, projectGeneratedFormFile)
	data := generatedFormSource(f.design)
	if renvoos.WriteFile(path, data, 0644) != nil {
		f.output.SetMessage("Could not update "+projectGeneratedFormFile+".", false)
		return
	}
	f.lastBuildOK = false
	// The generated file participates in semantic completion for user code.
	// Drop the whole snapshot so newly added controls and handlers are visible
	// immediately instead of retaining the previous generated source.
	f.analysis = editorAnalysisSession{}
	f.requestEditorAnalysis()
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
	data, err := renvoos.ReadFile(path)
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
	if renvoos.WriteFile(path, data, 0644) != nil {
		f.output.SetMessage("Could not create event handler in "+projectUserFormFile+".", false)
		return
	}
	if f.currentPath == path && (f.editor.Document == nil || !f.editor.Document.Dirty()) {
		f.editor.SetDocument(ide.NewDocument(data))
		f.syncEditorFrame()
	}
}

func ensureUserGraphicsImport(data []byte) []byte {
	const importPath = `"renvo.dev/std/graphics"`
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
	f.beginProjectBuild(false)
}

func (f *MainForm) beginProjectBuild(runAfter bool) {
	if f.buildSession != nil {
		f.output.SetMessage("A build is already in progress.", true)
		return
	}
	f.saveCurrentFile()
	f.output.SetMessage("Building the project…", true)
	f.lastBuildOK = false
	f.buildRunAfter = runAfter
	f.buildSession = beginCompileIDEProject(f.root, f.projectOutput, f.selectedTarget, f.env)
	if f.window != nil {
		f.window.SetTimer(projectBuildTimerID, 0)
		return
	}
	for f.buildSession != nil {
		f.stepProjectBuild()
	}
}

func (f *MainForm) runProject() {
	f.beginProjectBuild(true)
}

func (f *MainForm) stepProjectBuild() {
	if f == nil || f.buildSession == nil {
		return
	}
	done, result := f.buildSession.Step()
	if !done {
		if f.window != nil {
			f.window.SetTimer(projectBuildTimerID, 0)
		}
		return
	}
	runAfter := f.buildRunAfter
	f.buildSession = nil
	f.buildRunAfter = false
	f.lastBuildOK = result.ok
	f.output.SetMessage(result.message, result.ok)
	if !runAfter || !result.ok {
		return
	}
	if f.selectedTarget != defaultIDETarget() {
		f.output.SetMessage("Build succeeded for "+f.selectedTarget+". Run is available only for "+defaultIDETarget()+" on this host.", true)
		return
	}
	launched := launchIDEProject(f.projectOutput, f.root)
	f.output.SetMessage(launched.message, launched.ok)
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
	return []string{"darwin/arm64", "windows/amd64", "windows/386", "windows/arm64", "browser/wasm32"}
}

func workspaceProjectOutput(root, target string) string {
	name := projectOutputFile
	if workspaceHasPrefix(target, "windows/") {
		name += ".exe"
	} else if target == "wasi/wasm32" {
		name += ".wasm"
	} else if target == "browser/wasm32" {
		name += ".html"
	}
	return workspaceJoinPath(root, name)
}

// Dispatch keeps the working editor model and the surrounding status chrome
// synchronized without coupling editor commands to this particular shell.
func (f *MainForm) Dispatch(event graphics.Event) {
	if event.Type == graphics.EventTimer && event.TimerID == projectBuildTimerID {
		f.stepProjectBuild()
	}
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
