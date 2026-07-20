package main

import (
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

const workspaceAppBarHeight = 38
const workspacePaneHeaderHeight = 30
const workspaceStatusHeight = 24
const workspaceOutputHeight = 88
const designerGridSize = 10
const designerPaletteWidth = 140
const designerPaletteColumns = 4
const designerPaletteItemSize = 32
const designerPaletteItemStep = 33
const designerPalettePadding = 4
const workspacePropertyTitleHeight = 38
const workspacePropertyRowHeight = 32
const workspacePropertyFieldHeight = 24
const workspaceCodeTabWidth = 150
const workspaceDesignerTabWidth = 170
const workspaceTargetX = 112
const workspaceTargetWidth = 156
const workspaceBuildX = 276
const workspaceBuildWidth = 68
const workspaceRunX = 352
const workspaceRunWidth = 62
const workspaceTargetMenuPadding = 4
const workspaceTargetMenuRowHeight = 25

var workspaceWhite = graphics.RGBA(255, 255, 255, 255)
var workspaceRaised = graphics.RGBA(241, 243, 246, 255)
var workspaceCanvas = graphics.RGBA(250, 251, 253, 255)
var workspaceBorder = graphics.RGBA(218, 222, 228, 255)
var workspaceText = graphics.RGBA(28, 31, 36, 255)
var workspaceMuted = graphics.RGBA(97, 103, 113, 255)
var workspaceBlue = graphics.RGBA(25, 118, 210, 255)
var workspaceBlueLight = graphics.RGBA(226, 239, 255, 255)
var workspaceAccentText = graphics.RGBA(255, 255, 255, 255)
var workspaceField = graphics.RGBA(252, 252, 253, 255)
var workspaceGrid = graphics.RGBA(225, 229, 235, 255)
var workspaceDanger = graphics.RGBA(176, 55, 48, 255)
var workspaceDangerBackground = graphics.RGBA(255, 241, 239, 255)
var workspaceSuccess = graphics.RGBA(34, 137, 72, 255)

func applyWorkspaceTheme(theme forms.Theme) {
	workspaceWhite = theme.Surface
	workspaceRaised = theme.SurfaceRaised
	workspaceCanvas = theme.Window
	workspaceBorder = theme.Border
	workspaceText = theme.Text
	workspaceMuted = theme.MutedText
	workspaceBlue = theme.Accent
	workspaceBlueLight = theme.Selection
	workspaceAccentText = theme.AccentText
	workspaceField = theme.Field
	if int(theme.Window.R)+int(theme.Window.G)+int(theme.Window.B) < 384 {
		workspaceGrid = graphics.RGBA(62, 70, 84, 255)
		workspaceDanger = graphics.RGBA(244, 112, 104, 255)
		workspaceDangerBackground = graphics.RGBA(82, 43, 46, 255)
		workspaceSuccess = graphics.RGBA(91, 190, 118, 255)
	} else {
		workspaceGrid = graphics.RGBA(216, 222, 230, 255)
		workspaceDanger = graphics.RGBA(176, 55, 48, 255)
		workspaceDangerBackground = graphics.RGBA(255, 241, 239, 255)
		workspaceSuccess = graphics.RGBA(34, 137, 72, 255)
	}
}

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
	bodyHeight := frameHeight - workspacePaneHeaderHeight
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
	control.SetAccessibilityRole(forms.AccessibilityRoleGroup)
	control.SetAccessibilityName("Build toolbar")
	control.AccessibilityChildren = control.accessibilityChildren
	control.AccessibilityPerform = control.accessibilityPerform
	control.Paint = control.paint
	control.PointerDown = control.pointerDown
	return control
}

func (c *workspaceAppBar) pointerDown(x, y graphics.Scalar) {
	if x >= workspaceTargetX && x < workspaceTargetX+workspaceTargetWidth && c.OpenTargets != nil {
		c.OpenTargets()
		return
	}
	if x >= workspaceBuildX && x < workspaceBuildX+workspaceBuildWidth && c.Build != nil {
		c.Build()
		return
	}
	if x >= workspaceRunX && x < workspaceRunX+workspaceRunWidth && c.Run != nil {
		c.Run()
	}
}

func (c *workspaceAppBar) SetTarget(target string) {
	if c.target == target {
		return
	}
	c.target = target
	c.AccessibilityChildrenChanged()
	c.Invalidate()
}

func (c *workspaceAppBar) accessibilityChildren() []forms.AccessibilityNode {
	bounds := c.Bounds()
	baseID := c.AccessibilityID()
	return []forms.AccessibilityNode{
		{ID: baseID + "-target", Role: forms.AccessibilityRoleButton, Name: "Build target", Value: c.target, Bounds: graphics.R(bounds.MinX+workspaceTargetX, bounds.MinY+5, workspaceTargetWidth, 28), Actions: forms.AccessibilitySupportsInvoke},
		{ID: baseID + "-build", Role: forms.AccessibilityRoleButton, Name: "Build project", Bounds: graphics.R(bounds.MinX+workspaceBuildX, bounds.MinY+5, workspaceBuildWidth, 28), Actions: forms.AccessibilitySupportsInvoke},
		{ID: baseID + "-run", Role: forms.AccessibilityRoleButton, Name: "Run project", Bounds: graphics.R(bounds.MinX+workspaceRunX, bounds.MinY+5, workspaceRunWidth, 28), Actions: forms.AccessibilitySupportsInvoke},
	}
}

func (c *workspaceAppBar) accessibilityPerform(id string, action forms.AccessibilityAction, value string) bool {
	if action != forms.AccessibilityActionInvoke {
		return false
	}
	if id == c.AccessibilityID()+"-target" && c.OpenTargets != nil {
		c.OpenTargets()
		return true
	}
	if id == c.AccessibilityID()+"-build" && c.Build != nil {
		c.Build()
		return true
	}
	if id == c.AccessibilityID()+"-run" && c.Run != nil {
		c.Run()
		return true
	}
	return false
}

func (c *workspaceAppBar) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MaxY-1, bounds.Width(), 1), workspaceBorder)
	drawWorkspaceText(surface, c.font, bounds.MinX+16, bounds.MinY+25, "RENVO", workspaceBlue)
	targetBounds := graphics.R(bounds.MinX+workspaceTargetX, bounds.MinY+5, workspaceTargetWidth, 28)
	surface.FillRect(targetBounds, workspaceField)
	surface.StrokeRect(targetBounds, 1, workspaceBorder)
	drawWorkspaceText(surface, c.font, targetBounds.MinX+10, targetBounds.MinY+19, c.target, workspaceText)
	drawChevron(surface, targetBounds.MaxX-18, targetBounds.MinY+12, true, workspaceMuted)
	buildBounds := graphics.R(bounds.MinX+workspaceBuildX, bounds.MinY+5, workspaceBuildWidth, 28)
	surface.FillRect(buildBounds, workspaceBlueLight)
	drawWorkspaceText(surface, c.font, buildBounds.MinX+13, buildBounds.MinY+19, "BUILD", workspaceBlue)
	runBounds := graphics.R(bounds.MinX+workspaceRunX, bounds.MinY+5, workspaceRunWidth, 28)
	surface.FillRect(runBounds, workspaceBlue)
	drawRunIcon(surface, runBounds.MinX+11, runBounds.MinY+8, workspaceAccentText)
	drawWorkspaceText(surface, c.font, runBounds.MinX+27, runBounds.MinY+19, "RUN", workspaceAccentText)
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
	menu.SetAccessibilityRole(forms.AccessibilityRoleList)
	menu.SetAccessibilityName("Build targets")
	menu.AccessibilityChildren = menu.accessibilityChildren
	menu.AccessibilityPerform = menu.accessibilityPerform
	menu.Paint = menu.paint
	menu.PointerDown = menu.pointerDown
	return menu
}

func (c *workspaceTargetMenu) accessibilityChildren() []forms.AccessibilityNode {
	nodes := make([]forms.AccessibilityNode, 0, len(c.targets))
	bounds := c.Bounds()
	for i := 0; i < len(c.targets); i++ {
		nodes = append(nodes, forms.AccessibilityNode{
			ID:      c.AccessibilityID() + "-target-" + workspaceDecimal(i+1),
			Role:    forms.AccessibilityRoleListItem,
			Name:    c.targets[i],
			Bounds:  graphics.R(bounds.MinX, bounds.MinY+workspaceTargetMenuPadding+graphics.Scalar(i*workspaceTargetMenuRowHeight), bounds.Width(), workspaceTargetMenuRowHeight),
			Actions: forms.AccessibilitySupportsInvoke,
		})
	}
	return nodes
}

func (c *workspaceTargetMenu) accessibilityPerform(id string, action forms.AccessibilityAction, value string) bool {
	index, ok := workspaceAccessibilityIndex(id, c.AccessibilityID()+"-target-")
	if !ok || action != forms.AccessibilityActionInvoke || index < 0 || index >= len(c.targets) || c.Select == nil {
		return false
	}
	c.Select(c.targets[index])
	return true
}

func (c *workspaceTargetMenu) pointerDown(x, y graphics.Scalar) {
	if y < workspaceTargetMenuPadding {
		return
	}
	row := int(y-workspaceTargetMenuPadding) / workspaceTargetMenuRowHeight
	if row >= 0 && row < len(c.targets) && c.Select != nil {
		c.Select(c.targets[row])
	}
}

func (c *workspaceTargetMenu) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.StrokeRect(bounds, 1, workspaceBorder)
	for i := 0; i < len(c.targets); i++ {
		y := bounds.MinY + workspaceTargetMenuPadding + graphics.Scalar(i*workspaceTargetMenuRowHeight)
		drawWorkspaceText(surface, c.font, bounds.MinX+10, y+17, c.targets[i], workspaceText)
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
	control.SetAccessibilityName("Explorer pane")
	control.Paint = control.paint
	return control
}

func (c *workspaceExplorerFrame) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.FillRect(graphics.R(bounds.MaxX-1, bounds.MinY, 1, bounds.Height()), workspaceBorder)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY+workspacePaneHeaderHeight-1, bounds.Width(), 1), workspaceBorder)
	drawWorkspaceText(surface, c.font, bounds.MinX+12, bounds.MinY+21, "EXPLORER", workspaceText)
}

type workspaceEditorFrame struct {
	forms.Control
	font         *graphics.Font
	fileName     string
	dirty        bool
	line         int
	column       int
	diagnostic   string
	ShowDesigner func()
}

func (c *workspaceEditorFrame) SetDiagnostic(message string) {
	if c == nil || c.diagnostic == message {
		return
	}
	c.diagnostic = message
	if c.Form() != nil {
		bounds := c.Bounds()
		c.Form().Invalidate(graphics.R(bounds.MinX, bounds.MaxY-workspaceStatusHeight, bounds.Width(), workspaceStatusHeight))
	}
}

func newWorkspaceEditorFrame(font *graphics.Font) *workspaceEditorFrame {
	control := &workspaceEditorFrame{font: font, fileName: "main.go", line: 1, column: 1}
	control.Control = *forms.NewControl()
	control.SetTabStop(false)
	control.SetBackground(workspaceWhite)
	control.SetAccessibilityName("Editor pane")
	control.AccessibilityChildren = control.accessibilityChildren
	control.AccessibilityPerform = control.accessibilityPerform
	control.Paint = control.paint
	control.PointerDown = control.pointerDown
	return control
}

func (c *workspaceEditorFrame) accessibilityChildren() []forms.AccessibilityNode {
	bounds := c.Bounds()
	return []forms.AccessibilityNode{
		{
			ID:      c.AccessibilityID() + "-designer",
			Role:    forms.AccessibilityRoleButton,
			Name:    "Designer view",
			Bounds:  graphics.R(bounds.MinX+workspaceCodeTabWidth, bounds.MinY, workspaceDesignerTabWidth, workspacePaneHeaderHeight),
			Actions: forms.AccessibilitySupportsInvoke,
		},
	}
}

func (c *workspaceEditorFrame) accessibilityPerform(id string, action forms.AccessibilityAction, value string) bool {
	if id != c.AccessibilityID()+"-designer" || action != forms.AccessibilityActionInvoke || c.ShowDesigner == nil {
		return false
	}
	c.ShowDesigner()
	return true
}

func (c *workspaceEditorFrame) pointerDown(x, y graphics.Scalar) {
	if y >= 0 && y < workspacePaneHeaderHeight && x >= workspaceCodeTabWidth && x < workspaceCodeTabWidth+workspaceDesignerTabWidth && c.ShowDesigner != nil {
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
	drawWorkspaceText(surface, c.font, bounds.MinX+12, statusY+17, "Ln "+workspaceDecimal(c.line)+", Col "+workspaceDecimal(c.column), workspaceMuted)
	if c.diagnostic != "" {
		messageWidth := bounds.Width() - 122
		if messageWidth < 0 {
			messageWidth = 0
		}
		surface.PushClipRect(graphics.R(bounds.MinX+114, statusY+1, messageWidth, workspaceStatusHeight-1))
		drawWorkspaceText(surface, c.font, bounds.MinX+122, statusY+17, "Error: "+c.diagnostic, workspaceDanger)
		surface.PopClip()
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
	hoverPalette     int
}

func newWorkspaceDesigner(font *graphics.Font) *workspaceDesigner {
	control := &workspaceDesigner{font: font, selected: -1, hoverPalette: -1}
	control.Control = *forms.NewControl()
	control.SetTabStop(true)
	control.SetBackground(workspaceCanvas)
	control.SetAccessibilityRole(forms.AccessibilityRoleGroup)
	control.SetAccessibilityName("Form designer")
	control.AccessibilityChildren = control.accessibilityChildren
	control.AccessibilityPerform = control.accessibilityPerform
	control.Paint = control.paint
	control.PointerDown = control.pointerDown
	control.PointerMove = control.pointerMove
	control.PointerUp = control.pointerUp
	control.PointerLeave = control.pointerLeave
	control.KeyDown = control.keyDown
	return control
}

func (c *workspaceDesigner) SetDesign(design *formDesign) {
	c.design = design
	if design == nil || c.selected >= len(design.controls) {
		c.selected = -1
	}
	c.AccessibilityChildrenChanged()
	c.Invalidate()
}

func (c *workspaceDesigner) InvalidatePreview() {
	if c.design == nil || c.Form() == nil {
		return
	}
	layout := calculateDesignerPreview(designerCanvasBounds(graphics.R(0, 0, c.Bounds().Width(), c.Bounds().Height())), c.design)
	c.invalidateLocal(workspaceExpandRect(layout.outer, 7))
}

func (c *workspaceDesigner) SetSelection(index int) {
	if c.design == nil || index < -1 || index >= len(c.design.controls) || c.selected == index {
		return
	}
	c.invalidateSelection(c.selected)
	c.selected = index
	c.AccessibilityChildrenChanged()
	c.invalidateSelection(c.selected)
}

func (c *workspaceDesigner) pointerDown(x, y graphics.Scalar) {
	if c.selected >= 0 && x >= c.Bounds().Width()-34 && y >= 0 && y < workspacePaneHeaderHeight {
		c.deleteSelection()
		return
	}
	if y >= 0 && y < workspacePaneHeaderHeight && x >= 0 && x < workspaceCodeTabWidth && c.ShowCode != nil {
		c.ShowCode()
		return
	}
	palette := designerPaletteIndexAt(graphics.R(0, 0, c.Bounds().Width(), c.Bounds().Height()), x, y)
	if palette >= 0 {
		if c.AddControl != nil {
			c.AddControl(designerControlKinds[palette])
		}
		return
	}
	if x < designerPaletteWidth && y >= workspacePaneHeaderHeight {
		return
	}
	if c.design == nil {
		return
	}
	layout := calculateDesignerPreview(designerCanvasBounds(graphics.R(0, 0, c.Bounds().Width(), c.Bounds().Height())), c.design)
	if c.selected >= 0 {
		selectedBounds := designerControlBoundsAt(layout, c.design, c.selected)
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
		if workspacePointInRect(x, y, designerControlBoundsAt(layout, c.design, i)) {
			selected = i
			break
		}
	}
	c.SetSelection(selected)
	if c.SelectionChanged != nil {
		c.SelectionChanged(selected)
	}
	if selected >= 0 {
		if designerDockValue(c.design.controls[selected].dock) != designerDockNone {
			return
		}
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
	if c.dragMode == 0 {
		hover := designerPaletteIndexAt(graphics.R(0, 0, c.Bounds().Width(), c.Bounds().Height()), x, y)
		if hover != c.hoverPalette {
			c.invalidatePaletteHover(c.hoverPalette)
			c.hoverPalette = hover
			c.invalidatePaletteHover(c.hoverPalette)
		}
	}
	if c.design == nil || c.selected < 0 || c.dragMode == 0 {
		return
	}
	layout := calculateDesignerPreview(designerCanvasBounds(graphics.R(0, 0, c.Bounds().Width(), c.Bounds().Height())), c.design)
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
	c.AccessibilityChildrenChanged()
	newBounds := designerControlBounds(layout, next)
	c.invalidateLocal(workspaceExpandRect(workspaceUnionRect(oldBounds, newBounds), 6))
}

func (c *workspaceDesigner) pointerLeave() {
	if c.hoverPalette >= 0 {
		c.invalidatePaletteHover(c.hoverPalette)
		c.hoverPalette = -1
	}
}

func (c *workspaceDesigner) invalidatePaletteHover(index int) {
	if index < 0 || index >= len(designerControlKinds) || c.Form() == nil {
		return
	}
	bounds := graphics.R(0, 0, c.Bounds().Width(), c.Bounds().Height())
	c.invalidateLocal(designerPaletteItemBounds(bounds, index))
	c.invalidateLocal(designerPaletteTooltipBounds(c.font, bounds, index))
}

func (c *workspaceDesigner) accessibilityChildren() []forms.AccessibilityNode {
	baseID := c.AccessibilityID()
	bounds := c.Bounds()
	nodes := make([]forms.AccessibilityNode, 0, len(designerControlKinds)+4)
	nodes = append(nodes, forms.AccessibilityNode{ID: baseID + "-code", Role: forms.AccessibilityRoleButton, Name: "Code view", Bounds: graphics.R(bounds.MinX, bounds.MinY, workspaceCodeTabWidth, workspacePaneHeaderHeight), Actions: forms.AccessibilitySupportsInvoke})
	for i := 0; i < len(designerControlKinds); i++ {
		nodes = append(nodes, forms.AccessibilityNode{
			ID:      baseID + "-palette-" + workspaceDecimal(i+1),
			Role:    forms.AccessibilityRoleButton,
			Name:    "Add " + designerControlNames[i],
			Value:   designerControlKinds[i],
			Bounds:  designerPaletteItemBounds(bounds, i),
			Actions: forms.AccessibilitySupportsInvoke,
		})
	}
	if c.design != nil {
		canvas := designerCanvasBounds(bounds)
		layout := calculateDesignerPreview(canvas, c.design)
		for i := 0; i < len(c.design.controls); i++ {
			control := c.design.controls[i]
			name := control.text
			if name == "" {
				name = control.name
			}
			value := ""
			if control.kind == designerTextBox || control.kind == designerTextArea {
				value = control.text
			}
			checkable := control.kind == designerCheckBox || control.kind == designerRadioButton
			nodes = append(nodes, forms.AccessibilityNode{
				ID:          baseID + "-control-" + workspaceDecimal(i+1),
				Role:        designerAccessibilityRole(control.kind),
				Name:        name,
				Description: "Designer control " + control.name,
				Value:       value,
				Bounds:      designerControlBoundsAt(layout, c.design, i),
				Actions:     forms.AccessibilitySupportsInvoke,
				Checkable:   checkable,
				Checked:     checkable && control.checked,
				Selectable:  true,
				Selected:    i == c.selected,
			})
		}
	}
	if c.selected >= 0 && bounds.Width() >= 780 {
		nodes = append(nodes, forms.AccessibilityNode{ID: baseID + "-delete", Role: forms.AccessibilityRoleButton, Name: "Delete selected control", Bounds: graphics.R(bounds.MaxX-32, bounds.MinY+2, 26, 25), Actions: forms.AccessibilitySupportsInvoke})
	}
	return nodes
}

func (c *workspaceDesigner) accessibilityPerform(id string, action forms.AccessibilityAction, value string) bool {
	if action != forms.AccessibilityActionInvoke {
		return false
	}
	baseID := c.AccessibilityID()
	if id == baseID+"-code" && c.ShowCode != nil {
		c.ShowCode()
		return true
	}
	if id == baseID+"-delete" {
		c.deleteSelection()
		return true
	}
	if index, ok := workspaceAccessibilityIndex(id, baseID+"-palette-"); ok {
		if index >= 0 && index < len(designerControlKinds) && c.AddControl != nil {
			c.AddControl(designerControlKinds[index])
			return true
		}
	}
	if index, ok := workspaceAccessibilityIndex(id, baseID+"-control-"); ok && c.design != nil && index >= 0 && index < len(c.design.controls) {
		c.SetSelection(index)
		if c.SelectionChanged != nil {
			c.SelectionChanged(index)
		}
		return true
	}
	return false
}

func designerAccessibilityRole(kind string) forms.AccessibilityRole {
	if kind == designerButton {
		return forms.AccessibilityRoleButton
	}
	if kind == designerTextBox || kind == designerTextArea || kind == designerNumericUpDown {
		return forms.AccessibilityRoleTextBox
	}
	if kind == designerCheckBox {
		return forms.AccessibilityRoleCheckBox
	}
	if kind == designerRadioButton {
		return forms.AccessibilityRoleRadioButton
	}
	if kind == designerPictureBox {
		return forms.AccessibilityRoleImage
	}
	if kind == designerLabel {
		return forms.AccessibilityRoleLabel
	}
	if kind == designerComboBox || kind == designerListBox || kind == designerListView || kind == designerTabControl {
		return forms.AccessibilityRoleList
	}
	if kind == designerTreeView {
		return forms.AccessibilityRoleTree
	}
	if kind == designerProgressBar || kind == designerStatusBar {
		return forms.AccessibilityRoleStatus
	}
	if kind == designerMenuBar {
		return forms.AccessibilityRoleMenuBar
	}
	return forms.AccessibilityRoleGroup
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
	if c.selected >= 0 {
		deleteBounds := graphics.R(bounds.MaxX-32, bounds.MinY+2, 26, 25)
		surface.FillRect(deleteBounds, workspaceDangerBackground)
		drawDeleteIcon(surface, deleteBounds.MinX+6, deleteBounds.MinY+5, workspaceDanger)
	}
	canvas := designerCanvasBounds(bounds)
	surface.FillRect(canvas, workspaceCanvas)
	drawWorkspaceGrid(surface, canvas)
	c.drawForm(surface, canvas)
	drawDesignerPalette(surface, c.font, bounds, c.hoverPalette)
}

type workspaceOutput struct {
	forms.Control
	font    *graphics.Font
	message string
	ok      bool
}

func newWorkspaceOutput(font *graphics.Font) *workspaceOutput {
	control := &workspaceOutput{font: font, ok: true}
	control.Control = *forms.NewControl()
	control.SetTabStop(false)
	control.SetBackground(workspaceWhite)
	control.SetAccessibilityRole(forms.AccessibilityRoleStatus)
	control.SetAccessibilityName("Build output")
	control.AccessibilityValue = control.accessibilityValue
	control.Paint = control.paint
	return control
}

func (c *workspaceOutput) SetMessage(message string, ok bool) {
	if c.message == message && c.ok == ok {
		return
	}
	c.message = message
	c.ok = ok
	c.AccessibilityChanged()
	c.Invalidate()
}

func (c *workspaceOutput) accessibilityValue() string { return c.message }

func (c *workspaceOutput) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, bounds.Width(), 1), workspaceBorder)
	surface.FillRect(graphics.R(bounds.MaxX-1, bounds.MinY, 1, bounds.Height()), workspaceBorder)
	drawWorkspaceText(surface, c.font, bounds.MinX+12, bounds.MinY+21, "OUTPUT", workspaceText)
	if c.message == "" {
		return
	}
	color := workspaceDanger
	if c.ok {
		color = workspaceSuccess
	}
	textBounds := graphics.R(bounds.MinX+8, bounds.MinY+29, bounds.Width()-16, bounds.Height()-33)
	surface.PushClipRect(textBounds)
	message := wrapWorkspaceOutput(c.font, c.message, textBounds.Width()-8, 3)
	drawWorkspaceText(surface, c.font, bounds.MinX+12, bounds.MinY+49, message, color)
	surface.PopClip()
}

func wrapWorkspaceOutput(font *graphics.Font, text string, maxWidth graphics.Scalar, maxLines int) string {
	if font == nil || text == "" || maxWidth <= 0 || maxLines <= 0 {
		return ""
	}
	out := ""
	lines := 0
	for len(text) > 0 && lines < maxLines {
		lineEnd := 0
		for lineEnd < len(text) && text[lineEnd] != '\n' {
			lineEnd++
		}
		line := text[:lineEnd]
		consumed := lineEnd
		if graphics.MeasureText(font, line).Width > maxWidth {
			fit := workspaceOutputFit(font, line, maxWidth)
			if fit == 0 {
				fit = workspaceOutputNextUTF8(line, 0)
			}
			breakAt := fit
			for i := 0; i < fit; i++ {
				if line[i] == ' ' || line[i] == '\t' {
					breakAt = i
				}
			}
			if breakAt == 0 {
				breakAt = fit
			}
			line = line[:breakAt]
			consumed = breakAt
			for consumed < len(text) && (text[consumed] == ' ' || text[consumed] == '\t') {
				consumed++
			}
		} else if consumed < len(text) && text[consumed] == '\n' {
			consumed++
		}
		if out != "" {
			out += "\n"
		}
		out += line
		lines++
		text = text[consumed:]
	}
	return out
}

func workspaceOutputFit(font *graphics.Font, text string, maxWidth graphics.Scalar) int {
	fit := 0
	for fit < len(text) {
		next := workspaceOutputNextUTF8(text, fit)
		if graphics.MeasureText(font, text[:next]).Width > maxWidth {
			break
		}
		fit = next
	}
	return fit
}

func workspaceOutputNextUTF8(text string, at int) int {
	if at >= len(text) {
		return len(text)
	}
	width := 1
	first := text[at]
	if first&0xe0 == 0xc0 {
		width = 2
	} else if first&0xf0 == 0xe0 {
		width = 3
	} else if first&0xf8 == 0xf0 {
		width = 4
	}
	if at+width > len(text) {
		return at + 1
	}
	return at + width
}

func (c *workspaceDesigner) drawForm(surface *graphics.Surface, canvas graphics.Rect) {
	if c.design == nil {
		return
	}
	layout := calculateDesignerPreview(canvas, c.design)
	surface.FillRect(graphics.R(layout.outer.MinX+3, layout.outer.MinY+4, layout.outer.Width(), layout.outer.Height()), workspaceBorder)
	surface.FillRect(layout.outer, workspaceWhite)
	surface.StrokeRect(layout.outer, 1, workspaceBorder)
	surface.FillRect(graphics.R(layout.outer.MinX, layout.client.MinY-1, layout.outer.Width(), 1), workspaceBorder)
	drawCenteredWorkspaceText(surface, c.font, graphics.R(layout.outer.MinX, layout.outer.MinY, layout.outer.Width(), layout.client.MinY-layout.outer.MinY), "MainForm", workspaceText)
	for i := 0; i < len(c.design.controls); i++ {
		control := c.design.controls[i]
		bounds := designerControlBoundsAt(layout, c.design, i)
		if control.kind == designerButton {
			surface.FillRect(bounds, workspaceBlue)
			drawCenteredWorkspaceText(surface, c.font, bounds, control.text, workspaceAccentText)
		} else if control.kind == designerTextBox || control.kind == designerTextArea || control.kind == designerComboBox || control.kind == designerNumericUpDown {
			surface.FillRect(bounds, workspaceWhite)
			surface.StrokeRect(bounds, 1, workspaceBorder)
			value := control.text
			if control.kind == designerNumericUpDown {
				value = workspaceDecimal(control.value)
			}
			drawWorkspaceText(surface, c.font, bounds.MinX+6, bounds.MinY+c.font.Metrics.Ascent+4, value, workspaceText)
			if control.kind == designerComboBox || control.kind == designerNumericUpDown {
				surface.FillRect(graphics.R(bounds.MaxX-25, bounds.MinY, 25, bounds.Height()), workspaceField)
				drawChevron(surface, bounds.MaxX-17, bounds.MinY+bounds.Height()/2-2, true, workspaceMuted)
			}
		} else if control.kind == designerCheckBox {
			drawDesignerChoice(surface, c.font, bounds, control.text, control.checked, false)
		} else if control.kind == designerRadioButton {
			drawDesignerChoice(surface, c.font, bounds, control.text, control.checked, true)
		} else if control.kind == designerPictureBox {
			surface.FillRect(bounds, workspaceRaised)
			surface.StrokeRect(bounds, 1, workspaceBorder)
			surface.DrawLine(graphics.Point{X: bounds.MinX + 5, Y: bounds.MaxY - 6}, graphics.Point{X: bounds.MaxX - 5, Y: bounds.MinY + 6}, 1, workspaceMuted)
			surface.DrawLine(graphics.Point{X: bounds.MinX + 5, Y: bounds.MinY + 6}, graphics.Point{X: bounds.MaxX - 5, Y: bounds.MaxY - 6}, 1, workspaceMuted)
		} else if control.kind == designerPanel {
			surface.FillRect(bounds, workspaceRaised)
			surface.StrokeRect(bounds, 1, workspaceBorder)
		} else if control.kind == designerListBox || control.kind == designerListView || control.kind == designerTreeView {
			surface.FillRect(bounds, workspaceWhite)
			surface.StrokeRect(bounds, 1, workspaceBorder)
			if control.kind == designerListView {
				surface.FillRect(graphics.R(bounds.MinX+1, bounds.MinY+1, bounds.Width()-2, 22), workspaceField)
			}
			indent := graphics.Scalar(7)
			if control.kind == designerTreeView {
				indent = 20
			}
			drawWorkspaceText(surface, c.font, bounds.MinX+indent, bounds.MinY+c.font.Metrics.Ascent+7, control.text, workspaceText)
		} else if control.kind == designerTabControl {
			surface.FillRect(bounds, workspaceWhite)
			tab := graphics.R(bounds.MinX, bounds.MinY, bounds.Width()/2, bounds.Height())
			surface.FillRect(tab, workspaceBlueLight)
			surface.StrokeRect(bounds, 1, workspaceBorder)
			drawWorkspaceText(surface, c.font, tab.MinX+8, tab.MinY+c.font.Metrics.Ascent+6, control.text, workspaceText)
		} else if control.kind == designerProgressBar {
			surface.FillRect(bounds, workspaceField)
			amount := bounds.Width() * graphics.Scalar(control.value) / 100
			surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, amount, bounds.Height()), workspaceBlue)
			surface.StrokeRect(bounds, 1, workspaceBorder)
		} else if control.kind == designerSlider {
			y := bounds.MinY + bounds.Height()/2
			surface.FillRect(graphics.R(bounds.MinX+7, y-2, bounds.Width()-14, 4), workspaceBorder)
			x := bounds.MinX + 7 + (bounds.Width()-14)*graphics.Scalar(control.value)/100
			surface.FillEllipse(graphics.R(x-6, y-6, 12, 12), workspaceBlue)
		} else if control.kind == designerGroupBox {
			surface.StrokeRect(graphics.R(bounds.MinX, bounds.MinY+7, bounds.Width(), bounds.Height()-7), 1, workspaceBorder)
			drawWorkspaceText(surface, c.font, bounds.MinX+10, bounds.MinY+c.font.Metrics.Ascent, control.text, workspaceText)
		} else if control.kind == designerSplitContainer {
			surface.FillRect(bounds, workspaceWhite)
			surface.StrokeRect(bounds, 1, workspaceBorder)
			surface.FillRect(graphics.R(bounds.MinX+graphics.Scalar(control.value)-2, bounds.MinY, 5, bounds.Height()), workspaceBorder)
		} else if control.kind == designerToolBar || control.kind == designerStatusBar || control.kind == designerMenuBar {
			surface.FillRect(bounds, workspaceField)
			surface.StrokeRect(bounds, 1, workspaceBorder)
			label := control.text
			if control.kind == designerMenuBar {
				label = "File"
			}
			drawWorkspaceText(surface, c.font, bounds.MinX+9, bounds.MinY+(bounds.Height()-c.font.Metrics.Ascent-c.font.Metrics.Descent)/2+c.font.Metrics.Ascent, label, workspaceText)
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

func designerControlBoundsAt(layout designerPreview, design *formDesign, index int) graphics.Rect {
	if design == nil || index < 0 || index >= len(design.controls) {
		return graphics.Rect{}
	}
	control := design.controls[index]
	dock := designerDockValue(control.dock)
	if dock == designerDockNone {
		return designerControlBounds(layout, control)
	}
	left, top, right, bottom := 0, 0, design.width, design.height
	for i := 0; i < len(design.controls); i++ {
		candidate := design.controls[i]
		candidateDock := designerDockValue(candidate.dock)
		if candidateDock == designerDockNone || candidateDock == designerDockFill {
			continue
		}
		x, y, width, height := left, top, right-left, bottom-top
		if candidateDock == designerDockTop {
			height = candidate.height
			top += height
		} else if candidateDock == designerDockBottom {
			height = candidate.height
			y = bottom - height
			bottom -= height
		} else if candidateDock == designerDockLeft {
			width = candidate.width
			left += width
		} else if candidateDock == designerDockRight {
			width = candidate.width
			x = right - width
			right -= width
		}
		if i == index {
			return graphics.R(layout.client.MinX+graphics.Scalar(x)*layout.scale, layout.client.MinY+graphics.Scalar(y)*layout.scale, graphics.Scalar(width)*layout.scale, graphics.Scalar(height)*layout.scale)
		}
	}
	return graphics.R(layout.client.MinX+graphics.Scalar(left)*layout.scale, layout.client.MinY+graphics.Scalar(top)*layout.scale, graphics.Scalar(right-left)*layout.scale, graphics.Scalar(bottom-top)*layout.scale)
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
	layout := calculateDesignerPreview(designerCanvasBounds(graphics.R(0, 0, c.Bounds().Width(), c.Bounds().Height())), c.design)
	c.invalidateLocal(workspaceExpandRect(designerControlBoundsAt(layout, c.design, index), 6))
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
	control.SetAccessibilityRole(forms.AccessibilityRoleGroup)
	control.SetAccessibilityName("Properties")
	control.AccessibilityChildren = control.accessibilityChildren
	control.AccessibilityPerform = control.accessibilityPerform
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
	c.AccessibilityChildrenChanged()
	c.Invalidate()
}

func (c *workspaceInspector) SetSelection(index int) {
	if c.design == nil || index < -1 || index >= len(c.design.controls) || c.selected == index {
		return
	}
	c.commitEdit()
	c.selected = index
	c.active = ""
	c.AccessibilityChildrenChanged()
	c.Invalidate()
}

func (c *workspaceInspector) InvalidateProperties() {
	if c.Form() == nil {
		return
	}
	bounds := c.Bounds()
	c.AccessibilityChildrenChanged()
	c.Form().Invalidate(bounds)
}

func (c *workspaceInspector) accessibilityChildren() []forms.AccessibilityNode {
	properties := c.propertyNames()
	nodes := make([]forms.AccessibilityNode, 0, len(properties))
	bounds := c.Bounds()
	for i := 0; i < len(properties); i++ {
		name := properties[i]
		value := c.propertyValue(name)
		if c.active == name {
			value = c.editBuffer
		}
		role := forms.AccessibilityRoleTextBox
		actions := forms.AccessibilitySupportsFocus | forms.AccessibilitySupportsSetValue
		checkable, checked := false, false
		if name == "Checked" {
			role = forms.AccessibilityRoleCheckBox
			actions = forms.AccessibilitySupportsInvoke
			checkable, checked = true, value == "true"
		}
		nodes = append(nodes, forms.AccessibilityNode{
			ID:        c.AccessibilityID() + "-property-" + workspaceDecimal(i+1),
			Role:      role,
			Name:      name,
			Value:     value,
			Bounds:    graphics.R(bounds.MinX+70, bounds.MinY+workspacePaneHeaderHeight+workspacePropertyTitleHeight+graphics.Scalar(i*workspacePropertyRowHeight), bounds.Width()-82, workspacePropertyFieldHeight),
			Actions:   actions,
			Checkable: checkable,
			Checked:   checked,
		})
	}
	return nodes
}

func (c *workspaceInspector) accessibilityPerform(id string, action forms.AccessibilityAction, value string) bool {
	index, ok := workspaceAccessibilityIndex(id, c.AccessibilityID()+"-property-")
	properties := c.propertyNames()
	if !ok || index < 0 || index >= len(properties) {
		return false
	}
	c.active = properties[index]
	c.editBuffer = c.propertyValue(c.active)
	c.selectValue = true
	if action == forms.AccessibilityActionFocus {
		c.Focus()
		c.InvalidateProperties()
		return true
	}
	if action == forms.AccessibilityActionInvoke && c.active == "Checked" {
		if c.editBuffer == "true" {
			c.editBuffer = "false"
		} else {
			c.editBuffer = "true"
		}
		c.commitEdit()
		c.InvalidateProperties()
		return true
	}
	if action == forms.AccessibilityActionSetValue {
		c.editBuffer = value
		c.selectValue = false
		c.commitEdit()
		c.InvalidateProperties()
		return true
	}
	return false
}

func (c *workspaceInspector) pointerDown(x, y graphics.Scalar) {
	if c.design == nil || y < workspacePaneHeaderHeight {
		return
	}
	if y < workspacePaneHeaderHeight+workspacePropertyTitleHeight {
		return
	}
	properties := c.propertyNames()
	row := int(y-graphics.Scalar(workspacePaneHeaderHeight+workspacePropertyTitleHeight)) / workspacePropertyRowHeight
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
	if (c.active == "Click" || c.active == "Changed" || c.active == "Paint") && c.editBuffer == "" {
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
	} else if c.active == "Dock" && c.selected >= 0 {
		c.editBuffer = designerNextDockValue(c.editBuffer)
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
		} else if c.active == "Changed" && (c.editBuffer == "" || designerIdentifier(c.editBuffer)) && control.changeHandler != c.editBuffer {
			control.changeHandler = c.editBuffer
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
		} else if c.active == "Dock" && designerDockValue(c.editBuffer) == c.editBuffer && designerDockValue(control.dock) != c.editBuffer {
			control.dock = c.editBuffer
			if control.dock == designerDockNone {
				control.dock = ""
			}
			changed = true
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
				} else if c.active == "Value" && designerControlHasValue(control.kind) && control.value != value {
					control.value = value
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
	properties = append(properties, "X", "Y", "Width", "Height", "Dock")
	if designerControlHasChecked(control.kind) {
		properties = append(properties, "Checked")
	}
	if designerControlHasValue(control.kind) {
		properties = append(properties, "Value")
	}
	if control.kind == designerButton {
		properties = append(properties, "Click")
	}
	if designerControlHasChanged(control.kind) {
		properties = append(properties, "Changed")
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
	if name == "Dock" {
		return designerDockValue(control.dock)
	}
	if name == "Checked" {
		if control.checked {
			return "true"
		}
		return "false"
	}
	if name == "Value" {
		return workspaceDecimal(control.value)
	}
	if name == "Click" {
		return control.clickHandler
	}
	if name == "Changed" {
		return control.changeHandler
	}
	return control.paintHandler
}

func (c *workspaceInspector) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	surface.FillRect(bounds, workspaceWhite)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY+workspacePaneHeaderHeight-1, bounds.Width(), 1), workspaceBorder)
	drawWorkspaceText(surface, c.font, bounds.MinX+12, bounds.MinY+21, "PROPERTIES", workspaceText)
	surface.FillRect(graphics.R(bounds.MinX+8, bounds.MinY+workspacePaneHeaderHeight-2, 82, 2), workspaceBlue)
	c.drawProperties(surface, graphics.R(bounds.MinX+1, bounds.MinY+workspacePaneHeaderHeight, bounds.Width()-2, bounds.Height()-workspacePaneHeaderHeight))
}

func (c *workspaceInspector) drawProperties(surface *graphics.Surface, bounds graphics.Rect) {
	surface.PushClipRect(bounds)
	title := "MainForm"
	if c.design != nil && c.selected >= 0 && c.selected < len(c.design.controls) {
		title = c.design.controls[c.selected].name
	}
	drawWorkspaceText(surface, c.font, bounds.MinX+12, bounds.MinY+24, title, workspaceText)
	properties := c.propertyNames()
	y := bounds.MinY + workspacePropertyTitleHeight
	for i := 0; i < len(properties); i++ {
		value := c.propertyValue(properties[i])
		if c.active == properties[i] {
			value = c.editBuffer
		}
		kind := 0
		if properties[i] == "X" || properties[i] == "Y" || properties[i] == "Width" || properties[i] == "Height" {
			kind = 1
		} else if properties[i] == "Dock" {
			kind = 4
		}
		drawPropertyField(surface, c.font, bounds, y, properties[i], value, kind)
		if c.active == properties[i] {
			fieldX := bounds.MinX + 70
			surface.StrokeRect(graphics.R(fieldX, y, bounds.MaxX-fieldX-12, workspacePropertyFieldHeight), 2, workspaceBlue)
		}
		y += workspacePropertyRowHeight
	}
	surface.PopClip()
}

func drawPropertyField(surface *graphics.Surface, font *graphics.Font, bounds graphics.Rect, y graphics.Scalar, label, value string, kind int) {
	drawWorkspaceText(surface, font, bounds.MinX+12, y+17, label, workspaceText)
	fieldX := bounds.MinX + 70
	fieldWidth := bounds.MaxX - fieldX - 12
	if fieldWidth <= 0 {
		return
	}
	field := graphics.R(fieldX, y, fieldWidth, workspacePropertyFieldHeight)
	surface.FillRect(field, workspaceField)
	surface.StrokeRect(field, 1, workspaceBorder)
	textX := fieldX + 9
	if kind == 2 || kind == 3 {
		color := workspaceBlue
		if kind == 3 {
			color = workspaceAccentText
		}
		surface.FillRect(graphics.R(fieldX+8, y+7, 16, 13), color)
		surface.StrokeRect(graphics.R(fieldX+8, y+7, 16, 13), 1, workspaceBorder)
		textX = fieldX + 31
	}
	drawWorkspaceText(surface, font, textX, y+17, value, workspaceMuted)
	if kind == 1 && fieldWidth > 54 {
		surface.FillRect(graphics.R(field.MaxX-31, y, 1, workspacePropertyFieldHeight), workspaceBorder)
		drawWorkspaceText(surface, font, field.MaxX-23, y+17, "px", workspaceMuted)
	} else if kind == 4 && fieldWidth > 30 {
		surface.FillRect(graphics.R(field.MaxX-27, y, 1, workspacePropertyFieldHeight), workspaceBorder)
		points := []graphics.Point{{X: field.MaxX - 19, Y: y + 9}, {X: field.MaxX - 11, Y: y + 9}, {X: field.MaxX - 15, Y: y + 14}}
		surface.FillPolygon(points, graphics.FillNonZero, workspaceMuted)
	}
}

func designerCanvasBounds(bounds graphics.Rect) graphics.Rect {
	left := bounds.MinX + designerPaletteWidth
	top := bounds.MinY + workspacePaneHeaderHeight
	bottom := bounds.MaxY
	if left > bounds.MaxX {
		left = bounds.MaxX
	}
	if top > bottom {
		top = bottom
	}
	return graphics.R(left, top, bounds.MaxX-left, bottom-top)
}

func designerPaletteItemBounds(bounds graphics.Rect, index int) graphics.Rect {
	column := index % designerPaletteColumns
	row := index / designerPaletteColumns
	return graphics.R(bounds.MinX+designerPalettePadding+graphics.Scalar(column*designerPaletteItemStep), bounds.MinY+workspacePaneHeaderHeight+designerPalettePadding+graphics.Scalar(row*designerPaletteItemStep), designerPaletteItemSize, designerPaletteItemSize)
}

func designerPaletteIndexAt(bounds graphics.Rect, x, y graphics.Scalar) int {
	if x < bounds.MinX+designerPalettePadding || x >= bounds.MinX+designerPaletteWidth || y < bounds.MinY+workspacePaneHeaderHeight+designerPalettePadding || y >= bounds.MaxY {
		return -1
	}
	column := int(x-bounds.MinX-designerPalettePadding) / designerPaletteItemStep
	row := int(y-bounds.MinY-workspacePaneHeaderHeight-designerPalettePadding) / designerPaletteItemStep
	if column < 0 || column >= designerPaletteColumns || row < 0 {
		return -1
	}
	index := row*designerPaletteColumns + column
	if index < 0 || index >= len(designerControlKinds) || !workspacePointInRect(x, y, designerPaletteItemBounds(bounds, index)) {
		return -1
	}
	return index
}

func designerPaletteTooltipBounds(font *graphics.Font, bounds graphics.Rect, index int) graphics.Rect {
	if index < 0 || index >= len(designerControlNames) {
		return graphics.Rect{}
	}
	width := graphics.Scalar(len(designerControlNames[index])*8 + 18)
	if font != nil {
		width = graphics.MeasureText(font, designerControlNames[index]).Width + 18
	}
	item := designerPaletteItemBounds(bounds, index)
	y := item.MinY + 2
	if y+28 > bounds.MaxY {
		y = bounds.MaxY - 28
	}
	return graphics.R(bounds.MinX+designerPaletteWidth+7, y, width, 28)
}

func drawDesignerPalette(surface *graphics.Surface, font *graphics.Font, bounds graphics.Rect, hover int) {
	rail := graphics.R(bounds.MinX, bounds.MinY+workspacePaneHeaderHeight, designerPaletteWidth, bounds.Height()-workspacePaneHeaderHeight)
	surface.FillRect(rail, workspaceField)
	surface.FillRect(graphics.R(rail.MaxX-1, rail.MinY, 1, rail.Height()), workspaceBorder)
	surface.PushClipRect(rail)
	for i := 0; i < len(designerControlKinds); i++ {
		item := designerPaletteItemBounds(bounds, i)
		color := workspaceMuted
		if i == hover {
			surface.FillRect(item, workspaceBlueLight)
			surface.StrokeRect(item, 1, workspaceBlue)
			color = workspaceBlue
		}
		drawPaletteIcon(surface, item.MinX+3, item.MinY+3, i, color)
	}
	surface.PopClip()
	if hover >= 0 && hover < len(designerControlNames) {
		tooltip := designerPaletteTooltipBounds(font, bounds, hover)
		surface.FillRect(graphics.R(tooltip.MinX+2, tooltip.MinY+2, tooltip.Width(), tooltip.Height()), graphics.RGBA(53, 57, 64, 70))
		surface.FillRect(tooltip, graphics.RGBA(40, 43, 49, 255))
		surface.StrokeRect(tooltip, 1, graphics.RGBA(18, 20, 24, 255))
		drawWorkspaceText(surface, font, tooltip.MinX+9, tooltip.MinY+19, designerControlNames[hover], workspaceWhite)
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
	drawWorkspaceText(surface, font, bounds.MinX+9, bounds.MinY+20, placeholder, workspaceMuted)
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
	codeWidth := graphics.Scalar(workspaceCodeTabWidth)
	designWidth := graphics.Scalar(workspaceDesignerTabWidth)
	if codeWidth > bounds.Width() {
		codeWidth = bounds.Width()
	}
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY+workspacePaneHeaderHeight-1, bounds.Width(), 1), workspaceBorder)
	if !designerActive {
		surface.FillRect(graphics.R(bounds.MinX, bounds.MinY+workspacePaneHeaderHeight-2, codeWidth, 2), workspaceBlue)
	}
	drawWorkspaceText(surface, font, bounds.MinX+12, bounds.MinY+21, fileName, workspaceText)
	if dirty {
		surface.FillEllipse(graphics.R(bounds.MinX+codeWidth-17, bounds.MinY+12, 6, 6), workspaceMuted)
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
		surface.FillRect(graphics.R(designX, bounds.MinY+workspacePaneHeaderHeight-2, visibleDesignWidth, 2), workspaceBlue)
	}
	drawWorkspaceText(surface, font, designX+12, bounds.MinY+21, "MainForm [Design]", workspaceText)
}

func drawRunIcon(surface *graphics.Surface, x, y graphics.Scalar, color graphics.Color) {
	for column := 0; column < 10; column++ {
		halfHeight := graphics.Scalar(6 - column/2)
		centerY := y + 6
		surface.DrawLine(graphics.Point{X: x + graphics.Scalar(column), Y: centerY - halfHeight}, graphics.Point{X: x + graphics.Scalar(column), Y: centerY + halfHeight}, 1, color)
	}
}

func drawPaletteIcon(surface *graphics.Surface, x, y graphics.Scalar, kind int, color graphics.Color) {
	if kind < 0 || kind >= len(designerControlKinds) {
		return
	}
	forms.DrawControlIcon(surface, forms.Icon(int(forms.IconControlLabel)+kind), graphics.R(x, y, 26, 26), color, workspaceField, workspaceBlue)
}

func drawDeleteIcon(surface *graphics.Surface, x, y graphics.Scalar, color graphics.Color) {
	surface.StrokeRect(graphics.R(x+3, y+4, 10, 12), 1, color)
	surface.FillRect(graphics.R(x+1, y+3, 14, 2), color)
	surface.FillRect(graphics.R(x+5, y, 6, 2), color)
	surface.FillRect(graphics.R(x+6, y+7, 1, 6), color)
	surface.FillRect(graphics.R(x+10, y+7, 1, 6), color)
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

func workspaceAccessibilityIndex(id, prefix string) (int, bool) {
	if len(id) <= len(prefix) || !workspaceHasPrefix(id, prefix) {
		return -1, false
	}
	value := 0
	for i := len(prefix); i < len(id); i++ {
		if id[i] < '0' || id[i] > '9' {
			return -1, false
		}
		value = value*10 + int(id[i]-'0')
	}
	return value - 1, value > 0
}
