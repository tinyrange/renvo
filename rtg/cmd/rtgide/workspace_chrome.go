package main

import (
	"j5.nz/rtg/rtg/forms"
	"j5.nz/rtg/rtg/std/graphics"
)

const workspaceAppBarHeight = 46
const workspacePaneHeaderHeight = 36
const workspaceDesignerToolbarHeight = 42
const workspaceStatusHeight = 34
const workspaceOutputHeight = 112

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
	font  *graphics.Font
	Build func()
	Run   func()
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
	if x >= 170 && x < 242 && c.Build != nil {
		c.Build()
		return
	}
	if x >= 250 && x < 316 && c.Run != nil {
		c.Run()
	}
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
	buildBounds := graphics.R(bounds.MinX+170, bounds.MinY+9, 72, 28)
	surface.FillRect(buildBounds, workspaceBlueLight)
	drawWorkspaceText(surface, c.font, buildBounds.MinX+15, buildBounds.MinY+19, "BUILD", workspaceBlue)
	runBounds := graphics.R(bounds.MinX+250, bounds.MinY+9, 66, 28)
	surface.FillRect(runBounds, workspaceBlue)
	drawRunIcon(surface, runBounds.MinX+11, runBounds.MinY+8, workspaceWhite)
	drawWorkspaceText(surface, c.font, runBounds.MinX+28, runBounds.MinY+19, "RUN", workspaceWhite)
	buttonX := bounds.MaxX - 116
	surface.DrawLine(graphics.Point{X: buttonX, Y: bounds.MinY + 23}, graphics.Point{X: buttonX + 11, Y: bounds.MinY + 23}, 1, workspaceMuted)
	surface.StrokeRect(graphics.R(buttonX+43, bounds.MinY+17, 10, 10), 1, workspaceMuted)
	surface.DrawLine(graphics.Point{X: buttonX + 88, Y: bounds.MinY + 18}, graphics.Point{X: buttonX + 98, Y: bounds.MinY + 28}, 1, workspaceMuted)
	surface.DrawLine(graphics.Point{X: buttonX + 98, Y: bounds.MinY + 18}, graphics.Point{X: buttonX + 88, Y: bounds.MinY + 28}, 1, workspaceMuted)
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
	font     *graphics.Font
	ShowCode func()
	design   helloFormDesign
}

func newWorkspaceDesigner(font *graphics.Font) *workspaceDesigner {
	control := &workspaceDesigner{font: font, design: defaultHelloFormDesign()}
	control.Control = *forms.NewControl()
	control.SetTabStop(false)
	control.SetBackground(workspaceCanvas)
	control.Paint = control.paint
	control.PointerDown = control.pointerDown
	return control
}

func (c *workspaceDesigner) pointerDown(x, y graphics.Scalar) {
	if y >= 0 && y < workspacePaneHeaderHeight && x >= 0 && x < 170 && c.ShowCode != nil {
		c.ShowCode()
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
	drawDesignerToolbar(surface, bounds.MinX, headerBottom, bounds.Width())
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
	width := canvas.Width() - 74
	if width > graphics.Scalar(c.design.width) {
		width = graphics.Scalar(c.design.width)
	}
	if width < 170 {
		width = 170
	}
	height := width * graphics.Scalar(c.design.height) / graphics.Scalar(c.design.width)
	if height > canvas.Height()-24 {
		height = canvas.Height() - 24
	}
	if height < 150 {
		height = 150
	}
	x := canvas.MinX + (canvas.Width()-width)/2
	y := canvas.MinY + (canvas.Height()-height)/2
	selection := graphics.R(x, y, width, height)
	surface.StrokeRect(selection, 2, workspaceBlue)
	drawSelectionHandles(surface, selection)
	form := graphics.R(selection.MinX+16, selection.MinY+9, selection.Width()-32, selection.Height()-18)
	surface.FillRect(graphics.R(form.MinX+3, form.MinY+4, form.Width(), form.Height()), graphics.RGBA(227, 230, 235, 255))
	surface.FillRect(form, workspaceWhite)
	surface.StrokeRect(form, 1, workspaceBorder)
	headerHeight := graphics.Scalar(36)
	surface.FillRect(graphics.R(form.MinX, form.MinY+headerHeight-1, form.Width(), 1), workspaceBorder)
	drawCenteredWorkspaceText(surface, c.font, graphics.R(form.MinX, form.MinY, form.Width(), headerHeight), c.design.title, workspaceText)
	contentX := form.MinX + 20
	contentWidth := form.Width() - 40
	labelY := form.MinY + 79
	drawWorkspaceText(surface, c.font, contentX, labelY, c.design.messageText, workspaceText)
	buttonY := labelY + 26
	buttonHeight := graphics.Scalar(38)
	surface.FillRect(graphics.R(contentX, buttonY, contentWidth, buttonHeight), workspaceBlue)
	drawCenteredWorkspaceText(surface, c.font, graphics.R(contentX, buttonY, contentWidth, buttonHeight), c.design.buttonText, workspaceWhite)
}

type workspaceInspector struct {
	forms.Control
	font   *graphics.Font
	design helloFormDesign
}

func newWorkspaceInspector(font *graphics.Font) *workspaceInspector {
	control := &workspaceInspector{font: font, design: defaultHelloFormDesign()}
	control.Control = *forms.NewControl()
	control.SetTabStop(false)
	control.SetBackground(workspaceWhite)
	control.Paint = control.paint
	return control
}

func (c *workspaceInspector) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	paletteWidth := bounds.Width() * 39 / 100
	palette := graphics.R(bounds.MinX, bounds.MinY, paletteWidth, bounds.Height())
	properties := graphics.R(bounds.MinX+paletteWidth, bounds.MinY, bounds.Width()-paletteWidth, bounds.Height())
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY+workspacePaneHeaderHeight-1, bounds.Width(), 1), workspaceBorder)
	surface.FillRect(graphics.R(properties.MinX, bounds.MinY, 1, bounds.Height()), workspaceBorder)
	drawWorkspaceText(surface, c.font, palette.MinX+20, palette.MinY+24, "PALETTE", workspaceText)
	drawWorkspaceText(surface, c.font, properties.MinX+15, properties.MinY+24, "PROPERTIES", workspaceText)
	underlineWidth := graphics.Scalar(76)
	if underlineWidth > palette.Width()-16 {
		underlineWidth = palette.Width() - 16
	}
	if underlineWidth > 0 {
		surface.FillRect(graphics.R(palette.MinX+8, palette.MinY+workspacePaneHeaderHeight-2, underlineWidth, 2), workspaceOrange)
	}
	c.drawPalette(surface, graphics.R(palette.MinX, palette.MinY+workspacePaneHeaderHeight, palette.Width(), palette.Height()-workspacePaneHeaderHeight))
	c.drawProperties(surface, graphics.R(properties.MinX+1, properties.MinY+workspacePaneHeaderHeight, properties.Width()-1, properties.Height()-workspacePaneHeaderHeight))
}

func (c *workspaceInspector) drawPalette(surface *graphics.Surface, bounds graphics.Rect) {
	surface.PushClipRect(bounds)
	drawWorkspaceText(surface, c.font, bounds.MinX+18, bounds.MinY+25, "BASIC", workspaceText)
	items := []string{"Label", "Button", "Text Input", "Text Area", "Check Box", "Radio Button", "Image", "Container"}
	for i := 0; i < len(items); i++ {
		y := bounds.MinY + 55 + graphics.Scalar(i*39)
		drawPaletteIcon(surface, bounds.MinX+21, y-7, i, workspaceMuted)
		drawWorkspaceText(surface, c.font, bounds.MinX+42, y, items[i], workspaceText)
	}
	surface.PopClip()
}

func (c *workspaceInspector) drawProperties(surface *graphics.Surface, bounds graphics.Rect) {
	surface.PushClipRect(bounds)
	drawWorkspaceText(surface, c.font, bounds.MinX+16, bounds.MinY+28, "Button", workspaceText)
	y := bounds.MinY + 48
	drawPropertyField(surface, c.font, bounds, y, "ID", "button1", 0)
	y += 40
	drawPropertyField(surface, c.font, bounds, y, "Text", c.design.buttonText, 0)
	y += 40
	drawPropertyField(surface, c.font, bounds, y, "Width", "160", 1)
	y += 40
	drawPropertyField(surface, c.font, bounds, y, "Height", "36", 1)
	y += 40
	drawPropertyField(surface, c.font, bounds, y, "Background", "#1976D2", 2)
	y += 40
	drawPropertyField(surface, c.font, bounds, y, "Text Color", "#FFFFFF", 3)
	y += 40
	drawPropertyField(surface, c.font, bounds, y, "Radius", "4", 1)
	y += 42
	drawWorkspaceText(surface, c.font, bounds.MinX+16, y+18, "Alignment", workspaceText)
	fieldX := bounds.MinX + 78
	fieldWidth := bounds.MaxX - fieldX - 14
	if fieldWidth > 0 {
		surface.FillRect(graphics.R(fieldX, y, fieldWidth, 27), workspaceField)
		surface.StrokeRect(graphics.R(fieldX, y, fieldWidth, 27), 1, workspaceBorder)
		third := fieldWidth / 3
		surface.FillRect(graphics.R(fieldX+third, y+1, third, 25), workspaceBlueLight)
		drawWorkspaceText(surface, c.font, fieldX+12, y+19, "E", workspaceMuted)
		drawWorkspaceText(surface, c.font, fieldX+third+12, y+19, "=", workspaceBlue)
		drawWorkspaceText(surface, c.font, fieldX+third*2+12, y+19, "E", workspaceMuted)
	}
	y += 58
	drawChevron(surface, bounds.MinX+20, y+4, false, workspaceMuted)
	drawWorkspaceText(surface, c.font, bounds.MinX+34, y+9, "EVENTS", workspaceText)
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

func drawDesignerToolbar(surface *graphics.Surface, x, y, width graphics.Scalar) {
	selected := graphics.R(x+11, y+7, 29, 28)
	surface.FillRect(selected, workspaceBlueLight)
	points := []graphics.Point{{X: x + 20, Y: y + 14}, {X: x + 20, Y: y + 29}, {X: x + 25, Y: y + 24}, {X: x + 30, Y: y + 27}}
	surface.FillPolygon(points, graphics.FillNonZero, workspaceBlue)
	for i := 0; i < 4; i++ {
		iconX := x + 57 + graphics.Scalar(i*42)
		surface.StrokeRect(graphics.R(iconX, y+13, 11, 11), 1, workspaceMuted)
	}
	if width > 90 {
		for i := 0; i < 3; i++ {
			surface.FillEllipse(graphics.R(x+width-31+graphics.Scalar(i*5), y+20, 2, 2), workspaceMuted)
		}
	}
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
