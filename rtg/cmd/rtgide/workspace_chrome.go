package main

import (
	"j5.nz/rtg/rtg/forms"
	"j5.nz/rtg/rtg/std/graphics"
)

const workspaceAppBarHeight = 46
const workspacePaneHeaderHeight = 36
const workspaceDesignerToolbarHeight = 76
const workspaceStatusHeight = 34
const workspaceOutputHeight = 112
const designerGridSize = 10

var workspaceWhite = graphics.RGBA(255, 255, 255, 255)
var workspaceCanvas = graphics.RGBA(250, 251, 253, 255)
var workspaceBorder = graphics.RGBA(218, 222, 228, 255)
var workspaceText = graphics.RGBA(28, 31, 36, 255)
var workspaceMuted = graphics.RGBA(97, 103, 113, 255)
var workspaceBlue = graphics.RGBA(25, 118, 210, 255)
var workspaceBlueLight = graphics.RGBA(226, 239, 255, 255)
var workspacePurple = graphics.RGBA(126, 55, 221, 255)
var workspaceOrange = graphics.RGBA(236, 89, 19, 255)
var workspaceField = graphics.RGBA(252, 252, 253, 255)
var workspaceGrid = graphics.RGBA(225, 229, 235, 255)

type workspaceLayout struct {
	explorerFrame graphics.Rect
	editorFrame   graphics.Rect
	designer      graphics.Rect
	inspector     graphics.Rect
	output        graphics.Rect
	explorer      graphics.Rect
	editor        graphics.Rect
}

func calculateWorkspaceLayout(width, height int) workspaceLayout {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	explorerWidth := width * 183 / 1000
	documentRight := width * 756 / 1000
	if documentRight < explorerWidth {
		documentRight = explorerWidth
	}
	inspectorWidth := width - documentRight
	documentWidth := documentRight - explorerWidth
	frameHeight := height - workspaceAppBarHeight
	if frameHeight < 0 {
		frameHeight = 0
	}
	bodyHeight := frameHeight - workspacePaneHeaderHeight - workspaceStatusHeight
	if bodyHeight < 0 {
		bodyHeight = 0
	}
	explorerX := 0
	documentHeight := frameHeight - workspaceOutputHeight
	if documentHeight < workspacePaneHeaderHeight+workspaceStatusHeight {
		documentHeight = frameHeight
	}
	outputHeight := frameHeight - documentHeight
	documentBodyHeight := documentHeight - workspacePaneHeaderHeight - workspaceStatusHeight
	if documentBodyHeight < 0 {
		documentBodyHeight = 0
	}
	documentX := explorerWidth
	inspectorX := documentX + documentWidth
	return workspaceLayout{
		explorerFrame: rect(explorerX, workspaceAppBarHeight, explorerWidth, frameHeight),
		editorFrame:   rect(documentX, workspaceAppBarHeight, documentWidth, documentHeight),
		designer:      rect(documentX, workspaceAppBarHeight, documentWidth, documentHeight),
		inspector:     rect(inspectorX, workspaceAppBarHeight, inspectorWidth, frameHeight),
		output:        rect(documentX, workspaceAppBarHeight+documentHeight, documentWidth, outputHeight),
		explorer:      rect(explorerX, workspaceAppBarHeight+workspacePaneHeaderHeight, explorerWidth, bodyHeight),
		editor:        rect(documentX, workspaceAppBarHeight+workspacePaneHeaderHeight, documentWidth, documentBodyHeight),
	}
}

type workspaceAppBar struct {
	forms.Control
	font        *graphics.Font
	target      string
	Build       func()
	Run         func()
	OpenTargets func()
}

func newWorkspaceAppBar(font *graphics.Font) *workspaceAppBar {
	control := &workspaceAppBar{font: font}
	control.Control = *forms.NewControl()
	control.SetTabStop(false)
	control.SetBackground(workspaceWhite)
	control.Paint = control.paint
	control.PointerDown = control.pointerDown
	return control
}

func (c *workspaceAppBar) pointerDown(x, y graphics.Scalar) {
	if x >= 170 && x < 326 && c.OpenTargets != nil {
		c.OpenTargets()
		return
	}
	if x >= 334 && x < 406 && c.Build != nil {
		c.Build()
		return
	}
	if x >= 414 && x < 480 && c.Run != nil {
		c.Run()
	}
}

func (c *workspaceAppBar) SetTarget(target string) {
	if c.target == target {
		return
	}
	c.target = target
	c.Invalidate()
}

func (c *workspaceAppBar) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MaxY-1, bounds.Width(), 1), workspaceBorder)
	logo := graphics.R(bounds.MinX+16, bounds.MinY+13, 19, 19)
	surface.FillRect(logo, workspaceBlue)
	surface.DrawLine(graphics.Point{X: logo.MinX + 4, Y: logo.MaxY - 4}, graphics.Point{X: logo.MinX + 9, Y: logo.MinY + 4}, 2, workspaceWhite)
	surface.DrawLine(graphics.Point{X: logo.MinX + 9, Y: logo.MinY + 4}, graphics.Point{X: logo.MaxX - 4, Y: logo.MaxY - 4}, 2, workspaceWhite)
	drawWorkspaceText(surface, c.font, bounds.MinX+45, bounds.MinY+29, "MiniIDE", workspaceText)
	menuX := bounds.MinX + 124
	for i := 0; i < 3; i++ {
		y := bounds.MinY + 18 + graphics.Scalar(i*5)
		surface.DrawLine(graphics.Point{X: menuX, Y: y}, graphics.Point{X: menuX + 13, Y: y}, 1, workspaceMuted)
	}
	targetBounds := graphics.R(bounds.MinX+170, bounds.MinY+9, 156, 28)
	surface.FillRect(targetBounds, workspaceField)
	surface.StrokeRect(targetBounds, 1, workspaceBorder)
	drawWorkspaceText(surface, c.font, targetBounds.MinX+10, targetBounds.MinY+19, c.target, workspaceText)
	drawChevron(surface, targetBounds.MaxX-18, targetBounds.MinY+12, true, workspaceMuted)
	buildBounds := graphics.R(bounds.MinX+334, bounds.MinY+9, 72, 28)
	surface.FillRect(buildBounds, workspaceBlueLight)
	drawWorkspaceText(surface, c.font, buildBounds.MinX+15, buildBounds.MinY+19, "BUILD", workspaceBlue)
	runBounds := graphics.R(bounds.MinX+414, bounds.MinY+9, 66, 28)
	surface.FillRect(runBounds, workspaceBlue)
	drawRunIcon(surface, runBounds.MinX+11, runBounds.MinY+8, workspaceWhite)
	drawWorkspaceText(surface, c.font, runBounds.MinX+28, runBounds.MinY+19, "RUN", workspaceWhite)
	buttonX := bounds.MaxX - 116
	surface.DrawLine(graphics.Point{X: buttonX, Y: bounds.MinY + 23}, graphics.Point{X: buttonX + 11, Y: bounds.MinY + 23}, 1, workspaceMuted)
	surface.StrokeRect(graphics.R(buttonX+43, bounds.MinY+17, 10, 10), 1, workspaceMuted)
	surface.DrawLine(graphics.Point{X: buttonX + 88, Y: bounds.MinY + 18}, graphics.Point{X: buttonX + 98, Y: bounds.MinY + 28}, 1, workspaceMuted)
	surface.DrawLine(graphics.Point{X: buttonX + 98, Y: bounds.MinY + 18}, graphics.Point{X: buttonX + 88, Y: bounds.MinY + 28}, 1, workspaceMuted)
}

type workspaceTargetMenu struct {
	forms.Control
	font    *graphics.Font
	targets []string
	Select  func(target string)
}

func newWorkspaceTargetMenu(font *graphics.Font, targets []string) *workspaceTargetMenu {
	menu := &workspaceTargetMenu{font: font}
	menu.targets = append(menu.targets, targets...)
	menu.Control = *forms.NewControl()
	menu.SetTabStop(false)
	menu.SetBackground(workspaceWhite)
	menu.Paint = menu.paint
	menu.PointerDown = menu.pointerDown
	return menu
}

func (c *workspaceTargetMenu) pointerDown(x, y graphics.Scalar) {
	row := int(y-5) / 27
	if row >= 0 && row < len(c.targets) && c.Select != nil {
		c.Select(c.targets[row])
	}
}

func (c *workspaceTargetMenu) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.StrokeRect(bounds, 1, workspaceBorder)
	for i := 0; i < len(c.targets); i++ {
		y := bounds.MinY + 5 + graphics.Scalar(i*27)
		drawWorkspaceText(surface, c.font, bounds.MinX+12, y+18, c.targets[i], workspaceText)
	}
}

type workspaceExplorerFrame struct {
	forms.Control
	font *graphics.Font
}

func newWorkspaceExplorerFrame(font *graphics.Font) *workspaceExplorerFrame {
	control := &workspaceExplorerFrame{font: font}
	control.Control = *forms.NewControl()
	control.SetTabStop(false)
	control.SetBackground(workspaceWhite)
	control.Paint = control.paint
	return control
}

func (c *workspaceExplorerFrame) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.FillRect(graphics.R(bounds.MaxX-1, bounds.MinY, 1, bounds.Height()), workspaceBorder)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY+workspacePaneHeaderHeight-1, bounds.Width(), 1), workspaceBorder)
	drawWorkspaceText(surface, c.font, bounds.MinX+20, bounds.MinY+24, "EXPLORER", workspaceText)
	statusY := bounds.MaxY - workspaceStatusHeight
	surface.FillRect(graphics.R(bounds.MinX, statusY, bounds.Width(), 1), workspaceBorder)
	drawNewFileIcon(surface, bounds.MinX+22, statusY+10, workspaceMuted)
	drawSearchIcon(surface, bounds.MinX+67, statusY+17, workspaceMuted)
	for i := 0; i < 3; i++ {
		surface.FillEllipse(graphics.R(bounds.MinX+104+graphics.Scalar(i*6), statusY+16, 2, 2), workspaceMuted)
	}
}

type workspaceEditorFrame struct {
	forms.Control
	font         *graphics.Font
	fileName     string
	dirty        bool
	line         int
	column       int
	ShowDesigner func()
}

func newWorkspaceEditorFrame(font *graphics.Font) *workspaceEditorFrame {
	control := &workspaceEditorFrame{font: font, fileName: "main.go", line: 1, column: 1}
	control.Control = *forms.NewControl()
	control.SetTabStop(false)
	control.SetBackground(workspaceWhite)
	control.Paint = control.paint
	control.PointerDown = control.pointerDown
	return control
}

func (c *workspaceEditorFrame) pointerDown(x, y graphics.Scalar) {
	if y >= 0 && y < workspacePaneHeaderHeight && x >= 170 && x < 360 && c.ShowDesigner != nil {
		c.ShowDesigner()
	}
}

func (c *workspaceEditorFrame) SetDocumentState(path string, dirty bool, line, column int) {
	name := workspacePathBase(path)
	if name == "" {
		name = "main.go"
	}
	oldName := c.fileName
	oldDirty := c.dirty
	oldLine := c.line
	oldColumn := c.column
	c.fileName = name
	c.dirty = dirty
	c.line = line
	c.column = column
	if c.Form() == nil {
		return
	}
	bounds := c.Bounds()
	if oldName != name || oldDirty != dirty {
		c.Form().Invalidate(graphics.R(bounds.MinX, bounds.MinY, bounds.Width(), workspacePaneHeaderHeight))
	}
	if oldLine != line || oldColumn != column || oldDirty != dirty {
		c.Form().Invalidate(graphics.R(bounds.MinX, bounds.MaxY-workspaceStatusHeight, bounds.Width(), workspaceStatusHeight))
	}
}

func (c *workspaceEditorFrame) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.FillRect(graphics.R(bounds.MaxX-1, bounds.MinY, 1, bounds.Height()), workspaceBorder)
	drawDocumentTabs(surface, c.font, bounds, c.fileName, c.dirty, false)
	statusY := bounds.MaxY - workspaceStatusHeight
	surface.FillRect(graphics.R(bounds.MinX, statusY, bounds.Width(), 1), workspaceBorder)
	surface.StrokeEllipse(graphics.R(bounds.MinX+18, statusY+11, 10, 10), 2, workspaceBlue)
	drawWorkspaceText(surface, c.font, bounds.MinX+34, statusY+23, "Go 1.22", workspaceMuted)
	drawWorkspaceText(surface, c.font, bounds.MinX+142, statusY+23, "Ln "+workspaceDecimal(c.line)+", Col "+workspaceDecimal(c.column), workspaceMuted)
	drawWorkspaceText(surface, c.font, bounds.MinX+246, statusY+23, "Tab Size: 4", workspaceMuted)
	if bounds.Width() > 360 {
		drawWorkspaceText(surface, c.font, bounds.MaxX-78, statusY+23, "UTF-8    LF", workspaceMuted)
	}
}

type workspaceDesigner struct {
	forms.Control
	font             *graphics.Font
	ShowCode         func()
	Changed          func()
	SelectionChanged func(index int)
	AddControl       func(kind string)
	DeleteSelection  func(index int)
	design           *formDesign
	selected         int
	dragMode         int
	dragStartX       graphics.Scalar
	dragStartY       graphics.Scalar
	dragOriginal     designerControl
}

func newWorkspaceDesigner(font *graphics.Font) *workspaceDesigner {
	control := &workspaceDesigner{font: font, selected: -1}
	control.Control = *forms.NewControl()
	control.SetTabStop(true)
	control.SetBackground(workspaceCanvas)
	control.Paint = control.paint
	control.PointerDown = control.pointerDown
	control.PointerMove = control.pointerMove
	control.PointerUp = control.pointerUp
	control.KeyDown = control.keyDown
	return control
}

func (c *workspaceDesigner) SetDesign(design *formDesign) {
	c.design = design
	if design == nil || c.selected >= len(design.controls) {
		c.selected = -1
	}
	c.Invalidate()
}

func (c *workspaceDesigner) InvalidatePreview() {
	if c.design == nil || c.Form() == nil {
		return
	}
	layout := calculateDesignerPreview(graphics.R(0, workspacePaneHeaderHeight+workspaceDesignerToolbarHeight, c.Bounds().Width(), c.Bounds().Height()-workspacePaneHeaderHeight-workspaceDesignerToolbarHeight-workspaceStatusHeight), c.design)
	c.invalidateLocal(workspaceExpandRect(layout.outer, 7))
}

func (c *workspaceDesigner) SetSelection(index int) {
	if c.design == nil || index < -1 || index >= len(c.design.controls) || c.selected == index {
		return
	}
	c.invalidateSelection(c.selected)
	c.selected = index
	c.invalidateSelection(c.selected)
}

func (c *workspaceDesigner) pointerDown(x, y graphics.Scalar) {
	if y >= 0 && y < workspacePaneHeaderHeight && x >= 0 && x < 170 && c.ShowCode != nil {
		c.ShowCode()
		return
	}
	if y >= workspacePaneHeaderHeight && y < workspacePaneHeaderHeight+workspaceDesignerToolbarHeight {
		if c.selected >= 0 && c.Bounds().Width() >= 780 && x >= c.Bounds().Width()-72 && y < workspacePaneHeaderHeight+38 {
			c.deleteSelection()
			return
		}
		kind := designerPaletteKindAt(c.Bounds().Width(), x, y-workspacePaneHeaderHeight)
		if kind != "" && c.AddControl != nil {
			c.AddControl(kind)
		}
		return
	}
	if c.design == nil {
		return
	}
	layout := calculateDesignerPreview(graphics.R(0, workspacePaneHeaderHeight+workspaceDesignerToolbarHeight, c.Bounds().Width(), c.Bounds().Height()-workspacePaneHeaderHeight-workspaceDesignerToolbarHeight-workspaceStatusHeight), c.design)
	if c.selected >= 0 {
		selectedBounds := designerControlBounds(layout, c.design.controls[c.selected])
		if designerNear(x, selectedBounds.MaxX, 8) && designerNear(y, selectedBounds.MaxY, 8) {
			c.dragMode = 2
			c.dragStartX = x
			c.dragStartY = y
			c.dragOriginal = c.design.controls[c.selected]
			return
		}
	}
	selected := -1
	for i := len(c.design.controls) - 1; i >= 0; i-- {
		if workspacePointInRect(x, y, designerControlBounds(layout, c.design.controls[i])) {
			selected = i
			break
		}
	}
	c.SetSelection(selected)
	if c.SelectionChanged != nil {
		c.SelectionChanged(selected)
	}
	if selected >= 0 {
		c.dragMode = 1
		c.dragStartX = x
		c.dragStartY = y
		c.dragOriginal = c.design.controls[selected]
	}
}

func (c *workspaceDesigner) keyDown(event graphics.Event) {
	if event.Key == graphics.KeyDelete || event.Key == graphics.KeyBackspace && event.Modifiers&graphics.ModifierCommand != 0 {
		c.deleteSelection()
	}
}

func (c *workspaceDesigner) deleteSelection() {
	if c.selected >= 0 && c.DeleteSelection != nil {
		c.DeleteSelection(c.selected)
	}
}

func (c *workspaceDesigner) pointerMove(x, y graphics.Scalar) {
	if c.design == nil || c.selected < 0 || c.dragMode == 0 {
		return
	}
	layout := calculateDesignerPreview(graphics.R(0, workspacePaneHeaderHeight+workspaceDesignerToolbarHeight, c.Bounds().Width(), c.Bounds().Height()-workspacePaneHeaderHeight-workspaceDesignerToolbarHeight-workspaceStatusHeight), c.design)
	if layout.scale <= 0 {
		return
	}
	old := c.design.controls[c.selected]
	dx := int((x - c.dragStartX) / layout.scale)
	dy := int((y - c.dragStartY) / layout.scale)
	next := c.dragOriginal
	if c.dragMode == 1 {
		next.x = designerSnap(next.x + dx)
		next.y = designerSnap(next.y + dy)
		if next.x < 0 {
			next.x = 0
		}
		if next.y < 0 {
			next.y = 0
		}
		if next.x+next.width > c.design.width {
			next.x = c.design.width - next.width
		}
		if next.y+next.height > c.design.height {
			next.y = c.design.height - next.height
		}
	} else {
		next.width = designerSnap(next.width + dx)
		next.height = designerSnap(next.height + dy)
		if next.width < 16 {
			next.width = 16
		}
		if next.height < 16 {
			next.height = 16
		}
		if next.x+next.width > c.design.width {
			next.width = c.design.width - next.x
		}
		if next.y+next.height > c.design.height {
			next.height = c.design.height - next.y
		}
	}
	if old.x == next.x && old.y == next.y && old.width == next.width && old.height == next.height {
		return
	}
	oldBounds := designerControlBounds(layout, old)
	c.design.controls[c.selected] = next
	newBounds := designerControlBounds(layout, next)
	c.invalidateLocal(workspaceExpandRect(workspaceUnionRect(oldBounds, newBounds), 6))
}

func (c *workspaceDesigner) pointerUp(x, y graphics.Scalar) {
	if c.dragMode == 0 {
		return
	}
	changed := c.design != nil && c.selected >= 0 && c.design.controls[c.selected] != c.dragOriginal
	c.dragMode = 0
	if changed && c.Changed != nil {
		c.Changed()
	}
}

func (c *workspaceDesigner) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.FillRect(graphics.R(bounds.MaxX-1, bounds.MinY, 1, bounds.Height()), workspaceBorder)
	drawDocumentTabs(surface, c.font, bounds, "main_form.go", false, true)
	headerBottom := bounds.MinY + workspacePaneHeaderHeight
	surface.FillRect(graphics.R(bounds.MinX, headerBottom-1, bounds.Width(), 1), workspaceBorder)
	toolbarBottom := headerBottom + workspaceDesignerToolbarHeight
	surface.FillRect(graphics.R(bounds.MinX, toolbarBottom-1, bounds.Width(), 1), workspaceBorder)
	drawDesignerToolbar(surface, c.font, bounds.MinX, headerBottom, bounds.Width())
	if c.selected >= 0 && bounds.Width() >= 780 {
		deleteBounds := graphics.R(bounds.MaxX-72, headerBottom+6, 62, 27)
		surface.FillRect(deleteBounds, graphics.RGBA(255, 241, 239, 255))
		drawWorkspaceText(surface, c.font, deleteBounds.MinX+11, deleteBounds.MinY+18, "DELETE", graphics.RGBA(176, 55, 48, 255))
	}
	statusY := bounds.MaxY - workspaceStatusHeight
	canvas := graphics.R(bounds.MinX, toolbarBottom, bounds.Width(), statusY-toolbarBottom)
	surface.FillRect(canvas, workspaceCanvas)
	drawWorkspaceGrid(surface, canvas)
	c.drawForm(surface, canvas)
	surface.FillRect(graphics.R(bounds.MinX, statusY, bounds.Width(), 1), workspaceBorder)
}

type workspaceOutput struct {
	forms.Control
	font    *graphics.Font
	message string
	ok      bool
}

func newWorkspaceOutput(font *graphics.Font) *workspaceOutput {
	control := &workspaceOutput{font: font, message: "Ready. Build compiles the current project; Run launches the last successful build.", ok: true}
	control.Control = *forms.NewControl()
	control.SetTabStop(false)
	control.SetBackground(workspaceWhite)
	control.Paint = control.paint
	return control
}

func (c *workspaceOutput) SetMessage(message string, ok bool) {
	if c.message == message && c.ok == ok {
		return
	}
	c.message = message
	c.ok = ok
	c.Invalidate()
}

func (c *workspaceOutput) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, bounds.Width(), 1), workspaceBorder)
	surface.FillRect(graphics.R(bounds.MaxX-1, bounds.MinY, 1, bounds.Height()), workspaceBorder)
	drawWorkspaceText(surface, c.font, bounds.MinX+16, bounds.MinY+25, "OUTPUT", workspaceText)
	color := graphics.RGBA(176, 55, 48, 255)
	if c.ok {
		color = graphics.RGBA(34, 137, 72, 255)
	}
	surface.FillEllipse(graphics.R(bounds.MinX+79, bounds.MinY+17, 7, 7), color)
	surface.PushClipRect(graphics.R(bounds.MinX+12, bounds.MinY+35, bounds.Width()-24, bounds.Height()-39))
	drawWorkspaceText(surface, c.font, bounds.MinX+16, bounds.MinY+57, c.message, workspaceMuted)
	surface.PopClip()
}

func (c *workspaceDesigner) drawForm(surface *graphics.Surface, canvas graphics.Rect) {
	if c.design == nil {
		return
	}
	layout := calculateDesignerPreview(canvas, c.design)
	surface.FillRect(graphics.R(layout.outer.MinX+3, layout.outer.MinY+4, layout.outer.Width(), layout.outer.Height()), graphics.RGBA(227, 230, 235, 255))
	surface.FillRect(layout.outer, workspaceWhite)
	surface.StrokeRect(layout.outer, 1, workspaceBorder)
	surface.FillRect(graphics.R(layout.outer.MinX, layout.client.MinY-1, layout.outer.Width(), 1), workspaceBorder)
	drawCenteredWorkspaceText(surface, c.font, graphics.R(layout.outer.MinX, layout.outer.MinY, layout.outer.Width(), layout.client.MinY-layout.outer.MinY), "MainForm", workspaceText)
	for i := 0; i < len(c.design.controls); i++ {
		control := c.design.controls[i]
		bounds := designerControlBounds(layout, control)
		if control.kind == designerButton {
			surface.FillRect(bounds, workspaceBlue)
			drawCenteredWorkspaceText(surface, c.font, bounds, control.text, workspaceWhite)
		} else if control.kind == designerTextBox || control.kind == designerTextArea {
			surface.FillRect(bounds, workspaceWhite)
			surface.StrokeRect(bounds, 1, workspaceBorder)
			drawWorkspaceText(surface, c.font, bounds.MinX+6, bounds.MinY+c.font.Metrics.Ascent+4, control.text, workspaceText)
		} else if control.kind == designerCheckBox {
			drawDesignerChoice(surface, c.font, bounds, control.text, control.checked, false)
		} else if control.kind == designerRadioButton {
			drawDesignerChoice(surface, c.font, bounds, control.text, control.checked, true)
		} else if control.kind == designerPictureBox {
			surface.FillRect(bounds, graphics.RGBA(241, 243, 246, 255))
			surface.StrokeRect(bounds, 1, workspaceBorder)
			surface.DrawLine(graphics.Point{X: bounds.MinX + 5, Y: bounds.MaxY - 6}, graphics.Point{X: bounds.MaxX - 5, Y: bounds.MinY + 6}, 1, workspaceMuted)
			surface.DrawLine(graphics.Point{X: bounds.MinX + 5, Y: bounds.MinY + 6}, graphics.Point{X: bounds.MaxX - 5, Y: bounds.MaxY - 6}, 1, workspaceMuted)
		} else if control.kind == designerPanel {
			surface.FillRect(bounds, graphics.RGBA(247, 248, 250, 255))
			surface.StrokeRect(bounds, 1, workspaceBorder)
		} else {
			baseline := bounds.MinY + (bounds.Height()-c.font.Metrics.Ascent-c.font.Metrics.Descent)/2 + c.font.Metrics.Ascent
			drawWorkspaceText(surface, c.font, bounds.MinX, baseline, control.text, workspaceText)
		}
		if i == c.selected {
			surface.StrokeRect(bounds, 2, workspaceBlue)
			drawSelectionHandles(surface, bounds)
		}
	}
}

func drawDesignerChoice(surface *graphics.Surface, font *graphics.Font, bounds graphics.Rect, text string, checked, radio bool) {
	mark := graphics.R(bounds.MinX, bounds.MinY+(bounds.Height()-16)/2, 16, 16)
	if radio {
		surface.FillEllipse(mark, workspaceWhite)
		surface.StrokeEllipse(mark, 1, workspaceMuted)
		if checked {
			surface.FillEllipse(graphics.R(mark.MinX+4, mark.MinY+4, 8, 8), workspaceBlue)
		}
	} else {
		surface.FillRect(mark, workspaceWhite)
		surface.StrokeRect(mark, 1, workspaceMuted)
		if checked {
			surface.DrawLine(graphics.Point{X: mark.MinX + 3, Y: mark.MinY + 8}, graphics.Point{X: mark.MinX + 7, Y: mark.MinY + 12}, 2, workspaceBlue)
			surface.DrawLine(graphics.Point{X: mark.MinX + 7, Y: mark.MinY + 12}, graphics.Point{X: mark.MinX + 14, Y: mark.MinY + 3}, 2, workspaceBlue)
		}
	}
	baseline := bounds.MinY + (bounds.Height()-font.Metrics.Ascent-font.Metrics.Descent)/2 + font.Metrics.Ascent
	drawWorkspaceText(surface, font, bounds.MinX+23, baseline, text, workspaceText)
}

type designerPreview struct {
	outer  graphics.Rect
	client graphics.Rect
	scale  graphics.Scalar
}

func calculateDesignerPreview(canvas graphics.Rect, design *formDesign) designerPreview {
	if design == nil || design.width < 1 || design.height < 1 {
		return designerPreview{}
	}
	availableWidth := canvas.Width() - 64
	availableHeight := canvas.Height() - 52
	header := graphics.Scalar(30)
	scaleX := availableWidth / graphics.Scalar(design.width)
	scaleY := (availableHeight - header) / graphics.Scalar(design.height)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	if scale > 1 {
		scale = 1
	}
	if scale < 0.15 {
		scale = 0.15
	}
	width := graphics.Scalar(design.width) * scale
	height := graphics.Scalar(design.height) * scale
	outer := graphics.R(canvas.MinX+(canvas.Width()-width)/2, canvas.MinY+(canvas.Height()-height-header)/2, width, height+header)
	return designerPreview{outer: outer, client: graphics.R(outer.MinX, outer.MinY+header, width, height), scale: scale}
}

func designerControlBounds(layout designerPreview, control designerControl) graphics.Rect {
	return graphics.R(layout.client.MinX+graphics.Scalar(control.x)*layout.scale, layout.client.MinY+graphics.Scalar(control.y)*layout.scale, graphics.Scalar(control.width)*layout.scale, graphics.Scalar(control.height)*layout.scale)
}

func designerNear(a, b, distance graphics.Scalar) bool {
	return a >= b-distance && a <= b+distance
}

func designerSnap(value int) int {
	if value <= 0 {
		return 0
	}
	return (value + designerGridSize/2) / designerGridSize * designerGridSize
}

func workspacePointInRect(x, y graphics.Scalar, bounds graphics.Rect) bool {
	return x >= bounds.MinX && x < bounds.MaxX && y >= bounds.MinY && y < bounds.MaxY
}

func workspaceUnionRect(a, b graphics.Rect) graphics.Rect {
	if b.MinX < a.MinX {
		a.MinX = b.MinX
	}
	if b.MinY < a.MinY {
		a.MinY = b.MinY
	}
	if b.MaxX > a.MaxX {
		a.MaxX = b.MaxX
	}
	if b.MaxY > a.MaxY {
		a.MaxY = b.MaxY
	}
	return a
}

func workspaceExpandRect(rect graphics.Rect, amount graphics.Scalar) graphics.Rect {
	return graphics.R(rect.MinX-amount, rect.MinY-amount, rect.Width()+amount*2, rect.Height()+amount*2)
}

func (c *workspaceDesigner) invalidateLocal(rect graphics.Rect) {
	if c.Form() == nil {
		return
	}
	bounds := c.Bounds()
	c.Form().Invalidate(graphics.R(bounds.MinX+rect.MinX, bounds.MinY+rect.MinY, rect.Width(), rect.Height()))
}

func (c *workspaceDesigner) invalidateSelection(index int) {
	if c.design == nil || index < 0 || index >= len(c.design.controls) {
		return
	}
	layout := calculateDesignerPreview(graphics.R(0, workspacePaneHeaderHeight+workspaceDesignerToolbarHeight, c.Bounds().Width(), c.Bounds().Height()-workspacePaneHeaderHeight-workspaceDesignerToolbarHeight-workspaceStatusHeight), c.design)
	c.invalidateLocal(workspaceExpandRect(designerControlBounds(layout, c.design.controls[index]), 6))
}

type workspaceInspector struct {
	forms.Control
	font        *graphics.Font
	design      *formDesign
	selected    int
	active      string
	editBuffer  string
	selectValue bool
	Changed     func()
	CreateEvent func(handler string, paint bool)
}

func newWorkspaceInspector(font *graphics.Font) *workspaceInspector {
	control := &workspaceInspector{font: font, selected: -1}
	control.Control = *forms.NewControl()
	control.SetBackground(workspaceWhite)
	control.Paint = control.paint
	control.PointerDown = control.pointerDown
	control.TextInput = control.textInput
	control.KeyDown = control.keyDown
	return control
}

func (c *workspaceInspector) SetDesign(design *formDesign) {
	c.design = design
	if design == nil || c.selected >= len(design.controls) {
		c.selected = -1
	}
	c.active = ""
	c.Invalidate()
}

func (c *workspaceInspector) SetSelection(index int) {
	if c.design == nil || index < -1 || index >= len(c.design.controls) || c.selected == index {
		return
	}
	c.commitEdit()
	c.selected = index
	c.active = ""
	c.Invalidate()
}

func (c *workspaceInspector) InvalidateProperties() {
	if c.Form() == nil {
		return
	}
	bounds := c.Bounds()
	c.Form().Invalidate(bounds)
}

func (c *workspaceInspector) pointerDown(x, y graphics.Scalar) {
	if c.design == nil || y < workspacePaneHeaderHeight {
		return
	}
	if y < workspacePaneHeaderHeight+48 {
		return
	}
	properties := c.propertyNames()
	row := int(y-graphics.Scalar(workspacePaneHeaderHeight+48)) / 40
	if row < 0 || row >= len(properties) {
		c.commitEdit()
		c.active = ""
		c.InvalidateProperties()
		return
	}
	c.commitEdit()
	c.active = properties[row]
	c.editBuffer = c.propertyValue(c.active)
	c.selectValue = true
	if (c.active == "Click" || c.active == "Paint") && c.editBuffer == "" {
		base := "mainForm"
		if c.selected >= 0 {
			base = c.design.controls[c.selected].name
		}
		c.editBuffer = base + c.active
		c.selectValue = false
		c.commitEdit()
	} else if c.active == "Checked" && c.selected >= 0 {
		if c.design.controls[c.selected].checked {
			c.editBuffer = "false"
		} else {
			c.editBuffer = "true"
		}
		c.selectValue = false
		c.commitEdit()
	}
	c.InvalidateProperties()
}

func (c *workspaceInspector) textInput(text string) {
	if c.active == "" || text == "" {
		return
	}
	if c.selectValue {
		c.editBuffer = text
		c.selectValue = false
	} else {
		c.editBuffer = c.editBuffer + text
	}
	c.InvalidateProperties()
}

func (c *workspaceInspector) keyDown(event graphics.Event) {
	if c.active == "" {
		return
	}
	if event.Key == graphics.KeyEnter {
		c.commitEdit()
		c.active = ""
		c.InvalidateProperties()
	} else if event.Key == graphics.KeyEscape {
		c.active = ""
		c.InvalidateProperties()
	} else if event.Key == graphics.KeyBackspace {
		if c.selectValue {
			c.editBuffer = ""
			c.selectValue = false
		} else if len(c.editBuffer) > 0 {
			c.editBuffer = c.editBuffer[:len(c.editBuffer)-1]
		}
		c.InvalidateProperties()
	}
}

func (c *workspaceInspector) commitEdit() {
	if c.design == nil || c.active == "" {
		return
	}
	changed := false
	createdEvent := ""
	createdPaintEvent := false
	if c.selected < 0 {
		value, ok := designerParseInt(c.editBuffer)
		if c.active == "Paint" && (c.editBuffer == "" || designerIdentifier(c.editBuffer)) && c.design.paintHandler != c.editBuffer {
			c.design.paintHandler = c.editBuffer
			createdEvent = c.editBuffer
			createdPaintEvent = true
			changed = true
		} else if ok && value >= 100 {
			if c.active == "Width" && c.design.width != value {
				c.design.width = value
				changed = true
			} else if c.active == "Height" && c.design.height != value {
				c.design.height = value
				changed = true
			}
		}
	} else if c.selected < len(c.design.controls) {
		control := &c.design.controls[c.selected]
		if c.active == "Name" && designerIdentifier(c.editBuffer) && !c.nameInUse(c.editBuffer) && control.name != c.editBuffer {
			control.name = c.editBuffer
			changed = true
		} else if c.active == "Text" && control.text != c.editBuffer {
			control.text = c.editBuffer
			changed = true
		} else if c.active == "Click" && (c.editBuffer == "" || designerIdentifier(c.editBuffer)) && control.clickHandler != c.editBuffer {
			control.clickHandler = c.editBuffer
			createdEvent = c.editBuffer
			changed = true
		} else if c.active == "Paint" && (c.editBuffer == "" || designerIdentifier(c.editBuffer)) && control.paintHandler != c.editBuffer {
			control.paintHandler = c.editBuffer
			createdEvent = c.editBuffer
			createdPaintEvent = true
			changed = true
		} else if c.active == "Checked" && (c.editBuffer == "true" || c.editBuffer == "false") {
			checked := c.editBuffer == "true"
			if control.checked != checked {
				control.checked = checked
				changed = true
			}
		} else {
			value, ok := designerParseInt(c.editBuffer)
			if ok {
				if c.active == "X" && value >= 0 && value+control.width <= c.design.width && control.x != value {
					control.x = value
					changed = true
				} else if c.active == "Y" && value >= 0 && value+control.height <= c.design.height && control.y != value {
					control.y = value
					changed = true
				} else if c.active == "Width" && value >= 16 && control.x+value <= c.design.width && control.width != value {
					control.width = value
					changed = true
				} else if c.active == "Height" && value >= 16 && control.y+value <= c.design.height && control.height != value {
					control.height = value
					changed = true
				}
			}
		}
	}
	if changed && c.Changed != nil {
		c.Changed()
	}
	if createdEvent != "" && c.CreateEvent != nil {
		c.CreateEvent(createdEvent, createdPaintEvent)
	}
}

func (c *workspaceInspector) nameInUse(name string) bool {
	for i := 0; i < len(c.design.controls); i++ {
		if i != c.selected && c.design.controls[i].name == name {
			return true
		}
	}
	return false
}

func (c *workspaceInspector) propertyNames() []string {
	if c.design == nil || c.selected < 0 || c.selected >= len(c.design.controls) {
		return []string{"Width", "Height", "Paint"}
	}
	control := c.design.controls[c.selected]
	properties := []string{"Name"}
	if designerControlHasText(control.kind) {
		properties = append(properties, "Text")
	}
	properties = append(properties, "X", "Y", "Width", "Height")
	if designerControlHasChecked(control.kind) {
		properties = append(properties, "Checked")
	}
	if control.kind == designerButton {
		properties = append(properties, "Click")
	}
	properties = append(properties, "Paint")
	return properties
}

func (c *workspaceInspector) propertyValue(name string) string {
	if c.design == nil {
		return ""
	}
	if c.selected < 0 || c.selected >= len(c.design.controls) {
		if name == "Width" {
			return workspaceDecimal(c.design.width)
		}
		if name == "Height" {
			return workspaceDecimal(c.design.height)
		}
		return c.design.paintHandler
	}
	control := c.design.controls[c.selected]
	if name == "Name" {
		return control.name
	}
	if name == "Text" {
		return control.text
	}
	if name == "X" {
		return workspaceDecimal(control.x)
	}
	if name == "Y" {
		return workspaceDecimal(control.y)
	}
	if name == "Width" {
		return workspaceDecimal(control.width)
	}
	if name == "Height" {
		return workspaceDecimal(control.height)
	}
	if name == "Checked" {
		if control.checked {
			return "true"
		}
		return "false"
	}
	if name == "Click" {
		return control.clickHandler
	}
	return control.paintHandler
}

func (c *workspaceInspector) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY+workspacePaneHeaderHeight-1, bounds.Width(), 1), workspaceBorder)
	drawWorkspaceText(surface, c.font, bounds.MinX+16, bounds.MinY+24, "PROPERTIES", workspaceText)
	surface.FillRect(graphics.R(bounds.MinX+8, bounds.MinY+workspacePaneHeaderHeight-2, 92, 2), workspaceOrange)
	c.drawProperties(surface, graphics.R(bounds.MinX+1, bounds.MinY+workspacePaneHeaderHeight, bounds.Width()-2, bounds.Height()-workspacePaneHeaderHeight))
}

func (c *workspaceInspector) drawProperties(surface *graphics.Surface, bounds graphics.Rect) {
	surface.PushClipRect(bounds)
	title := "MainForm"
	if c.design != nil && c.selected >= 0 && c.selected < len(c.design.controls) {
		title = c.design.controls[c.selected].name
	}
	drawWorkspaceText(surface, c.font, bounds.MinX+16, bounds.MinY+28, title, workspaceText)
	properties := c.propertyNames()
	y := bounds.MinY + 48
	for i := 0; i < len(properties); i++ {
		value := c.propertyValue(properties[i])
		if c.active == properties[i] {
			value = c.editBuffer
		}
		kind := 0
		if properties[i] == "X" || properties[i] == "Y" || properties[i] == "Width" || properties[i] == "Height" {
			kind = 1
		}
		drawPropertyField(surface, c.font, bounds, y, properties[i], value, kind)
		if c.active == properties[i] {
			fieldX := bounds.MinX + 78
			surface.StrokeRect(graphics.R(fieldX, y, bounds.MaxX-fieldX-14, 27), 2, workspaceBlue)
		}
		y += 40
	}
	surface.PopClip()
}

func drawPropertyField(surface *graphics.Surface, font *graphics.Font, bounds graphics.Rect, y graphics.Scalar, label, value string, kind int) {
	drawWorkspaceText(surface, font, bounds.MinX+16, y+18, label, workspaceText)
	fieldX := bounds.MinX + 78
	fieldWidth := bounds.MaxX - fieldX - 14
	if fieldWidth <= 0 {
		return
	}
	field := graphics.R(fieldX, y, fieldWidth, 27)
	surface.FillRect(field, workspaceField)
	surface.StrokeRect(field, 1, workspaceBorder)
	textX := fieldX + 9
	if kind == 2 || kind == 3 {
		color := workspaceBlue
		if kind == 3 {
			color = workspaceWhite
		}
		surface.FillRect(graphics.R(fieldX+8, y+7, 16, 13), color)
		surface.StrokeRect(graphics.R(fieldX+8, y+7, 16, 13), 1, workspaceBorder)
		textX = fieldX + 31
	}
	drawWorkspaceText(surface, font, textX, y+18, value, workspaceMuted)
	if kind == 1 && fieldWidth > 54 {
		surface.FillRect(graphics.R(field.MaxX-31, y, 1, 27), workspaceBorder)
		drawWorkspaceText(surface, font, field.MaxX-23, y+18, "px", workspaceMuted)
	}
}

func drawDesignerToolbar(surface *graphics.Surface, font *graphics.Font, x, y, width graphics.Scalar) {
	selected := graphics.R(x+11, y+7, 29, 28)
	surface.FillRect(selected, workspaceBlueLight)
	points := []graphics.Point{{X: x + 20, Y: y + 14}, {X: x + 20, Y: y + 29}, {X: x + 25, Y: y + 24}, {X: x + 30, Y: y + 27}}
	surface.FillPolygon(points, graphics.FillNonZero, workspaceBlue)
	items := []string{"Label", "Button", "Text", "Text Area", "Check", "Radio", "Image", "Panel"}
	columns := designerPaletteColumns(width)
	for i := 0; i < len(items); i++ {
		column := i % columns
		row := i / columns
		itemX := x + 48 + graphics.Scalar(column*88)
		itemY := y + 5 + graphics.Scalar(row*33)
		drawPaletteIcon(surface, itemX+8, itemY+11, i, workspaceMuted)
		drawWorkspaceText(surface, font, itemX+25, itemY+20, items[i], workspaceText)
	}
}

func designerPaletteColumns(width graphics.Scalar) int {
	columns := int(width-48) / 88
	if columns < 1 {
		columns = 1
	}
	if columns > 8 {
		columns = 8
	}
	return columns
}

func designerPaletteKindAt(width, x, y graphics.Scalar) string {
	if x < 48 || y < 5 {
		return ""
	}
	columns := designerPaletteColumns(width)
	column := int(x-48) / 88
	row := int(y-5) / 33
	if column < 0 || column >= columns || row < 0 || row > 1 {
		return ""
	}
	index := row*columns + column
	kinds := []string{designerLabel, designerButton, designerTextBox, designerTextArea, designerCheckBox, designerRadioButton, designerPictureBox, designerPanel}
	if index < 0 || index >= len(kinds) {
		return ""
	}
	return kinds[index]
}

func drawWorkspaceGrid(surface *graphics.Surface, bounds graphics.Rect) {
	for y := bounds.MinY + 8; y < bounds.MaxY; y += 10 {
		for x := bounds.MinX + 8; x < bounds.MaxX; x += 10 {
			surface.FillRect(graphics.R(x, y, 1, 1), workspaceGrid)
		}
	}
}

func drawSelectionHandles(surface *graphics.Surface, bounds graphics.Rect) {
	middleX := (bounds.MinX + bounds.MaxX) / 2
	middleY := (bounds.MinY + bounds.MaxY) / 2
	drawSelectionHandle(surface, bounds.MinX, bounds.MinY)
	drawSelectionHandle(surface, middleX, bounds.MinY)
	drawSelectionHandle(surface, bounds.MaxX, bounds.MinY)
	drawSelectionHandle(surface, bounds.MinX, middleY)
	drawSelectionHandle(surface, bounds.MaxX, middleY)
	drawSelectionHandle(surface, bounds.MinX, bounds.MaxY)
	drawSelectionHandle(surface, middleX, bounds.MaxY)
	drawSelectionHandle(surface, bounds.MaxX, bounds.MaxY)
}

func drawSelectionHandle(surface *graphics.Surface, x, y graphics.Scalar) {
	handle := graphics.R(x-3, y-3, 7, 7)
	surface.FillRect(handle, workspaceWhite)
	surface.StrokeRect(handle, 1, workspaceBlue)
}

func drawMockField(surface *graphics.Surface, font *graphics.Font, bounds graphics.Rect, placeholder string) {
	surface.FillRect(bounds, workspaceWhite)
	surface.StrokeRect(bounds, 1, workspaceBorder)
	drawWorkspaceText(surface, font, bounds.MinX+9, bounds.MinY+20, placeholder, graphics.RGBA(157, 162, 171, 255))
}

func drawWorkspaceText(surface *graphics.Surface, font *graphics.Font, x, baseline graphics.Scalar, text string, color graphics.Color) {
	if font == nil || text == "" {
		return
	}
	surface.DrawText(font, graphics.Point{X: x, Y: baseline}, text, color)
}

func drawCenteredWorkspaceText(surface *graphics.Surface, font *graphics.Font, bounds graphics.Rect, text string, color graphics.Color) {
	metrics := graphics.MeasureText(font, text)
	x := bounds.MinX + (bounds.Width()-metrics.Width)/2
	baseline := bounds.MinY + (bounds.Height()-metrics.Height)/2 + font.Metrics.Ascent
	drawWorkspaceText(surface, font, x, baseline, text, color)
}

func drawDocumentTabs(surface *graphics.Surface, font *graphics.Font, bounds graphics.Rect, fileName string, dirty bool, designerActive bool) {
	codeWidth := graphics.Scalar(170)
	designWidth := graphics.Scalar(190)
	if codeWidth > bounds.Width() {
		codeWidth = bounds.Width()
	}
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY+workspacePaneHeaderHeight-1, bounds.Width(), 1), workspaceBorder)
	if !designerActive {
		surface.FillRect(graphics.R(bounds.MinX, bounds.MinY+workspacePaneHeaderHeight-2, codeWidth, 2), workspaceBlue)
	}
	drawGoIcon(surface, font, bounds.MinX+15, bounds.MinY+24)
	drawWorkspaceText(surface, font, bounds.MinX+39, bounds.MinY+24, fileName, workspaceText)
	if dirty {
		surface.FillEllipse(graphics.R(bounds.MinX+codeWidth-21, bounds.MinY+16, 6, 6), workspaceMuted)
	} else if codeWidth >= 42 {
		drawCloseIcon(surface, bounds.MinX+codeWidth-20, bounds.MinY+18, workspaceMuted)
	}
	if bounds.Width() <= codeWidth {
		return
	}
	designX := bounds.MinX + codeWidth
	visibleDesignWidth := designWidth
	if visibleDesignWidth > bounds.MaxX-designX {
		visibleDesignWidth = bounds.MaxX - designX
	}
	surface.FillRect(graphics.R(designX, bounds.MinY, 1, workspacePaneHeaderHeight), workspaceBorder)
	if designerActive {
		surface.FillRect(graphics.R(designX, bounds.MinY+workspacePaneHeaderHeight-2, visibleDesignWidth, 2), workspacePurple)
	}
	drawUIFileIcon(surface, designX+14, bounds.MinY+10)
	drawWorkspaceText(surface, font, designX+37, bounds.MinY+24, "MainForm [Design]", workspaceText)
}

func drawRunIcon(surface *graphics.Surface, x, y graphics.Scalar, color graphics.Color) {
	for column := 0; column < 10; column++ {
		halfHeight := graphics.Scalar(6 - column/2)
		centerY := y + 6
		surface.DrawLine(graphics.Point{X: x + graphics.Scalar(column), Y: centerY - halfHeight}, graphics.Point{X: x + graphics.Scalar(column), Y: centerY + halfHeight}, 1, color)
	}
}

func drawGoIcon(surface *graphics.Surface, font *graphics.Font, x, baseline graphics.Scalar) {
	drawWorkspaceText(surface, font, x, baseline, "go", workspaceBlue)
}

func drawUIFileIcon(surface *graphics.Surface, x, y graphics.Scalar) {
	surface.StrokeRect(graphics.R(x, y, 14, 16), 1, workspacePurple)
	surface.FillRect(graphics.R(x+4, y+5, 6, 2), workspacePurple)
	surface.FillRect(graphics.R(x+4, y+9, 6, 2), workspacePurple)
}

func drawCloseIcon(surface *graphics.Surface, x, y graphics.Scalar, color graphics.Color) {
	surface.DrawLine(graphics.Point{X: x, Y: y}, graphics.Point{X: x + 7, Y: y + 7}, 1, color)
	surface.DrawLine(graphics.Point{X: x + 7, Y: y}, graphics.Point{X: x, Y: y + 7}, 1, color)
}

func drawNewFileIcon(surface *graphics.Surface, x, y graphics.Scalar, color graphics.Color) {
	surface.StrokeRect(graphics.R(x, y, 12, 15), 1, color)
	surface.DrawLine(graphics.Point{X: x + 6, Y: y + 4}, graphics.Point{X: x + 6, Y: y + 12}, 1, color)
	surface.DrawLine(graphics.Point{X: x + 2, Y: y + 8}, graphics.Point{X: x + 10, Y: y + 8}, 1, color)
}

func drawSearchIcon(surface *graphics.Surface, x, y graphics.Scalar, color graphics.Color) {
	surface.StrokeEllipse(graphics.R(x, y-6, 10, 10), 1, color)
	surface.DrawLine(graphics.Point{X: x + 8, Y: y + 2}, graphics.Point{X: x + 13, Y: y + 7}, 1, color)
}

func drawPaletteIcon(surface *graphics.Surface, x, y graphics.Scalar, kind int, color graphics.Color) {
	if kind == 0 {
		surface.DrawLine(graphics.Point{X: x, Y: y + 12}, graphics.Point{X: x + 5, Y: y}, 1, color)
		surface.DrawLine(graphics.Point{X: x + 5, Y: y}, graphics.Point{X: x + 11, Y: y + 12}, 1, color)
		return
	}
	if kind == 4 {
		surface.StrokeRect(graphics.R(x, y, 13, 13), 1, color)
		surface.DrawLine(graphics.Point{X: x + 3, Y: y + 7}, graphics.Point{X: x + 6, Y: y + 10}, 1, color)
		surface.DrawLine(graphics.Point{X: x + 6, Y: y + 10}, graphics.Point{X: x + 11, Y: y + 3}, 1, color)
		return
	}
	if kind == 5 {
		surface.StrokeEllipse(graphics.R(x, y, 13, 13), 1, color)
		surface.FillEllipse(graphics.R(x+5, y+5, 3, 3), color)
		return
	}
	surface.StrokeRect(graphics.R(x, y, 13, 13), 1, color)
	if kind == 1 {
		surface.FillEllipse(graphics.R(x+5, y+5, 3, 3), color)
	}
}

func drawChevron(surface *graphics.Surface, x, y graphics.Scalar, expanded bool, color graphics.Color) {
	if expanded {
		surface.DrawLine(graphics.Point{X: x, Y: y}, graphics.Point{X: x + 4, Y: y + 4}, 1, color)
		surface.DrawLine(graphics.Point{X: x + 4, Y: y + 4}, graphics.Point{X: x + 8, Y: y}, 1, color)
		return
	}
	surface.DrawLine(graphics.Point{X: x, Y: y}, graphics.Point{X: x + 4, Y: y + 4}, 1, color)
	surface.DrawLine(graphics.Point{X: x + 4, Y: y + 4}, graphics.Point{X: x, Y: y + 8}, 1, color)
}

func workspacePathBase(path string) string {
	end := len(path)
	for end > 0 && (path[end-1] == '/' || path[end-1] == '\\') {
		end--
	}
	start := end
	for start > 0 && path[start-1] != '/' && path[start-1] != '\\' {
		start--
	}
	return path[start:end]
}

func workspaceDecimal(value int) string {
	if value <= 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	text := make([]byte, len(digits))
	for i := 0; i < len(digits); i++ {
		text[i] = digits[len(digits)-i-1]
	}
	return string(text)
}
