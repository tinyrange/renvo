package ide

import (
	"j5.nz/rtg/rtg/forms"
	"j5.nz/rtg/rtg/std/graphics"
)

var explorerBackground = graphics.RGBA(255, 255, 255, 255)
var editorBackground = graphics.RGBA(255, 255, 255, 255)
var borderColor = graphics.RGBA(214, 218, 224, 255)
var selectionColor = graphics.RGBA(226, 239, 255, 255)
var selectionTextColor = graphics.RGBA(35, 39, 47, 255)
var textColor = graphics.RGBA(35, 39, 47, 255)
var lineNumberColor = graphics.RGBA(132, 139, 150, 255)
var currentLineColor = graphics.RGBA(248, 250, 253, 255)
var keywordColor = graphics.RGBA(0, 82, 180, 255)
var builtinColor = graphics.RGBA(0, 118, 118, 255)
var stringColor = graphics.RGBA(156, 36, 33, 255)
var commentColor = graphics.RGBA(38, 128, 62, 255)
var numberColor = graphics.RGBA(108, 62, 176, 255)

const editorGutterWidth = 54
const editorVerticalScrollMargin = 3
const editorHorizontalScrollMargin = 4

// ExplorerControl renders and edits an Explorer model through a retained forms
// control. OpenFile is wired by generated form code to a user callback.
type ExplorerControl struct {
	forms.Control
	Model     *Explorer
	Font      *graphics.Font
	OpenFile  func(path string)
	scrollY   graphics.Scalar
	rowHeight int
	baseline  int
}

func NewExplorerControl(model *Explorer) *ExplorerControl {
	control := &ExplorerControl{Model: model}
	control.Control = *forms.NewControl()
	control.SetFont(graphics.NewBuiltinFont(2))
	control.SetBackground(explorerBackground)
	control.Paint = control.paint
	control.PointerDown = control.pointerDown
	control.PointerWheel = control.pointerWheel
	control.Click = control.activate
	control.KeyDown = control.keyDown
	return control
}

func (c *ExplorerControl) SetFont(font *graphics.Font) {
	if font == nil {
		font = graphics.NewBuiltinFont(2)
	}
	c.Font = font
	c.rowHeight = fontLineHeight(font) + 6
	c.baseline = fontPixelCeil(font.Metrics.Ascent) + 3
	c.Invalidate()
}

func (c *ExplorerControl) RowHeight() int { return c.rowHeight }

func (c *ExplorerControl) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	c.clampScroll()
	surface.FillRect(bounds, c.Background())
	rows := c.Model.Rows()
	visible := int(bounds.Height()) / c.rowHeight
	if visible < 1 {
		visible = 1
	}
	first := int(c.scrollY) / c.rowHeight
	offset := c.scrollY - graphics.Scalar(first*c.rowHeight)
	end := first + visible + 2
	if end > len(rows) {
		end = len(rows)
	}
	for i := first; i < end; i++ {
		y := bounds.MinY - offset + graphics.Scalar((i-first)*c.rowHeight)
		rowRect := graphics.R(bounds.MinX+1, y, bounds.Width()-2, graphics.Scalar(c.rowHeight))
		color := textColor
		if i == c.Model.SelectedIndex() {
			surface.FillRect(rowRect, selectionColor)
			color = selectionTextColor
		}
		node := rows[i].Node
		x := bounds.MinX + 16 + graphics.Scalar(rows[i].Depth*18)
		if node.Directory {
			drawExplorerChevron(surface, x, y+graphics.Scalar(c.rowHeight/2-3), node.Expanded, color)
			drawExplorerFolder(surface, x+15, y+graphics.Scalar(c.rowHeight/2-7))
		} else {
			drawExplorerFile(surface, c.Font, x+15, y+graphics.Scalar(c.rowHeight/2-8), node.Name)
		}
		surface.DrawText(c.Font, graphics.Point{X: x + 36, Y: y + graphics.Scalar(c.baseline)}, node.Name, color)
	}
	surface.FillRect(graphics.R(bounds.MaxX-1, bounds.MinY, 1, bounds.Height()), borderColor)
}

func drawExplorerChevron(surface *graphics.Surface, x, y graphics.Scalar, expanded bool, color graphics.Color) {
	if expanded {
		surface.DrawLine(graphics.Point{X: x, Y: y}, graphics.Point{X: x + 4, Y: y + 4}, 1, color)
		surface.DrawLine(graphics.Point{X: x + 4, Y: y + 4}, graphics.Point{X: x + 8, Y: y}, 1, color)
		return
	}
	surface.DrawLine(graphics.Point{X: x, Y: y}, graphics.Point{X: x + 4, Y: y + 4}, 1, color)
	surface.DrawLine(graphics.Point{X: x + 4, Y: y + 4}, graphics.Point{X: x, Y: y + 8}, 1, color)
}

func drawExplorerFolder(surface *graphics.Surface, x, y graphics.Scalar) {
	fill := graphics.RGBA(224, 228, 233, 255)
	stroke := graphics.RGBA(102, 109, 119, 255)
	surface.FillRect(graphics.R(x+1, y, 7, 3), fill)
	surface.FillRect(graphics.R(x, y+3, 16, 11), fill)
	surface.StrokeRect(graphics.R(x, y+3, 16, 11), 1, stroke)
}

func drawExplorerFile(surface *graphics.Surface, font *graphics.Font, x, y graphics.Scalar, name string) {
	if ideHasSuffix(name, ".go") {
		surface.DrawText(font, graphics.Point{X: x, Y: y + font.Metrics.Ascent + 2}, "go", graphics.RGBA(20, 105, 214, 255))
		return
	}
	color := graphics.RGBA(99, 106, 116, 255)
	if ideHasSuffix(name, ".ui") {
		color = graphics.RGBA(126, 55, 221, 255)
	}
	surface.StrokeRect(graphics.R(x+2, y, 12, 15), 1, color)
	if ideHasSuffix(name, ".png") || ideHasSuffix(name, ".jpg") {
		surface.FillEllipse(graphics.R(x+5, y+3, 3, 3), color)
		surface.DrawLine(graphics.Point{X: x + 4, Y: y + 12}, graphics.Point{X: x + 8, Y: y + 8}, 1, color)
		surface.DrawLine(graphics.Point{X: x + 8, Y: y + 8}, graphics.Point{X: x + 12, Y: y + 12}, 1, color)
	}
}

func ideHasSuffix(text, suffix string) bool {
	if len(suffix) > len(text) {
		return false
	}
	start := len(text) - len(suffix)
	for i := 0; i < len(suffix); i++ {
		if text[start+i] != suffix[i] {
			return false
		}
	}
	return true
}

func (c *ExplorerControl) pointerDown(x, y graphics.Scalar) {
	index := int(c.scrollY+y) / c.rowHeight
	rows := c.Model.Rows()
	if index < 0 || index >= len(rows) {
		return
	}
	old := c.Model.SelectedIndex()
	c.Model.Select(index)
	c.invalidateRow(old)
	c.invalidateRow(index)
}

func (c *ExplorerControl) pointerWheel(x, y graphics.Scalar) {
	old := c.scrollY
	c.scrollY -= y
	c.clampScroll()
	if old != c.scrollY {
		c.Invalidate()
	}
}

func (c *ExplorerControl) activate() {
	before := len(c.Model.Rows())
	path, ok := c.Model.ActivateSelected()
	if ok && c.OpenFile != nil {
		c.OpenFile(path)
	}
	if len(c.Model.Rows()) != before {
		c.Invalidate()
	}
}

func (c *ExplorerControl) keyDown(event graphics.Event) {
	old := c.Model.SelectedIndex()
	oldRows := len(c.Model.Rows())
	if event.Key == graphics.KeyUp {
		c.Model.Move(-1)
	} else if event.Key == graphics.KeyDown {
		c.Model.Move(1)
	} else if event.Key == graphics.KeyHome {
		c.Model.First()
	} else if event.Key == graphics.KeyEnd {
		c.Model.Last()
	} else if event.Key == graphics.KeyLeft {
		c.Model.CollapseOrParent()
	} else if event.Key == graphics.KeyRight {
		c.Model.ExpandOrChild()
	} else if event.Key == graphics.KeyEnter {
		c.activate()
		return
	} else {
		return
	}
	c.ensureSelectionVisible()
	if oldRows != len(c.Model.Rows()) {
		c.Invalidate()
	} else {
		c.invalidateRow(old)
		c.invalidateRow(c.Model.SelectedIndex())
	}
}

func (c *ExplorerControl) ensureSelectionVisible() {
	selected := c.Model.SelectedIndex()
	visible := int(c.Bounds().Height()) / c.rowHeight
	if visible < 1 {
		visible = 1
	}
	top := graphics.Scalar(selected * c.rowHeight)
	bottom := top + graphics.Scalar(c.rowHeight)
	if top < c.scrollY {
		c.scrollY = top
	}
	viewportBottom := c.scrollY + c.Bounds().Height()
	if bottom > viewportBottom {
		c.scrollY = bottom - c.Bounds().Height()
	}
	c.clampScroll()
}

func (c *ExplorerControl) clampScroll() {
	maximum := graphics.Scalar(len(c.Model.Rows())*c.rowHeight) - c.Bounds().Height()
	if maximum < 0.0 {
		maximum = 0.0
	}
	if c.scrollY < 0.0 {
		c.scrollY = 0.0
	}
	if c.scrollY > maximum {
		c.scrollY = maximum
	}
}

func (c *ExplorerControl) invalidateRow(index int) {
	if c.Form() == nil {
		return
	}
	bounds := c.Bounds()
	y := bounds.MinY + graphics.Scalar(index*c.rowHeight) - c.scrollY
	if y < bounds.MaxY && y+graphics.Scalar(c.rowHeight) > bounds.MinY {
		c.Form().Invalidate(graphics.R(bounds.MinX, y, bounds.Width(), graphics.Scalar(c.rowHeight)))
	}
}

// EditorControl renders and edits a Document. Save is an application callback
// wired by generated form code; the editor itself owns no filesystem policy.
type EditorControl struct {
	forms.Control
	Document       *Document
	Font           *graphics.Font
	Save           func()
	scrollY        graphics.Scalar
	scrollX        graphics.Scalar
	dragging       bool
	lineHeight     int
	characterWidth graphics.Scalar
	baseline       int
	syntaxScratch  []syntaxSpan
	syntaxStates   []goSyntaxState
}

func NewEditorControl(document *Document) *EditorControl {
	control := &EditorControl{Document: document}
	control.Control = *forms.NewControl()
	control.SetFont(graphics.NewBuiltinFont(2))
	control.SetBackground(editorBackground)
	control.Paint = control.paint
	control.PointerDown = control.pointerDown
	control.PointerMove = control.pointerMove
	control.PointerUp = control.pointerUp
	control.PointerWheel = control.pointerWheel
	control.TextInput = control.textInput
	control.KeyDown = control.keyDown
	return control
}

func (c *EditorControl) SetFont(font *graphics.Font) {
	if font == nil {
		font = graphics.NewBuiltinFont(2)
	}
	c.Font = font
	c.lineHeight = fontLineHeight(font) + 4
	c.characterWidth = fontCellWidth(font)
	c.baseline = fontPixelCeil(font.Metrics.Ascent) + 1
	c.Invalidate()
}

func (c *EditorControl) SetDocument(document *Document) {
	if document == nil {
		document = NewDocument(nil)
	}
	c.Document = document
	c.scrollY = 0.0
	c.scrollX = 0.0
	c.syntaxStates = nil
	c.rebuildSyntaxStates(0)
	c.Invalidate()
}

func (c *EditorControl) ScrollPosition() (int, int) {
	line := 0
	column := 0
	if c.lineHeight > 0 {
		line = int(c.scrollY) / c.lineHeight
	}
	if c.characterWidth > 0.0 {
		column = int(c.scrollX / c.characterWidth)
	}
	return line, column
}

func (c *EditorControl) VisibleGrid() (int, int) {
	visibleLines := int(c.Bounds().Height()) / c.lineHeight
	visibleColumns := int((c.Bounds().Width() - editorGutterWidth) / c.characterWidth)
	if visibleLines < 0 {
		visibleLines = 0
	}
	if visibleColumns < 0 {
		visibleColumns = 0
	}
	return visibleLines, visibleColumns
}

func (c *EditorControl) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	c.clampScroll()
	surface.FillRect(bounds, c.Background())
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, editorGutterWidth, bounds.Height()), explorerBackground)
	firstLine := int(c.scrollY) / c.lineHeight
	lineOffset := c.scrollY - graphics.Scalar(firstLine*c.lineHeight)
	visible := int(bounds.Height())/c.lineHeight + 2
	end := firstLine + visible
	if end > c.Document.LineCount() {
		end = c.Document.LineCount()
	}
	caretLine, caretColumn := c.Document.VisualPosition(c.Document.Caret)
	selectionStart, selectionEnd := c.Document.Selection()
	textClip := graphics.R(bounds.MinX+editorGutterWidth, bounds.MinY, bounds.Width()-editorGutterWidth, bounds.Height())
	state := goSyntaxNormal
	if firstLine >= 0 && firstLine < len(c.syntaxStates) {
		state = c.syntaxStates[firstLine]
	}
	for line := firstLine; line < end; line++ {
		y := bounds.MinY - lineOffset + graphics.Scalar((line-firstLine)*c.lineHeight)
		lineRect := graphics.R(bounds.MinX+editorGutterWidth, y, bounds.Width()-editorGutterWidth, graphics.Scalar(c.lineHeight))
		if line == caretLine && c.Focused() {
			surface.FillRect(lineRect, currentLineColor)
		}
		lineText := c.Document.LineText(line)
		lineStart := c.Document.Offset(line, 0)
		lineEnd := lineStart + len(lineText)
		selectedStart := selectionStart
		if selectedStart < lineStart {
			selectedStart = lineStart
		}
		selectedEnd := selectionEnd
		if selectedEnd > lineEnd {
			selectedEnd = lineEnd
		}
		if selectedEnd > selectedStart {
			_, startColumn := c.Document.VisualPosition(selectedStart)
			_, endColumn := c.Document.VisualPosition(selectedEnd)
			x := bounds.MinX + editorGutterWidth + graphics.Scalar(startColumn)*c.characterWidth - c.scrollX
			width := graphics.Scalar(endColumn-startColumn) * c.characterWidth
			surface.PushClipRect(textClip)
			surface.FillRect(graphics.R(x, y, width, graphics.Scalar(c.lineHeight)), selectionColor)
			surface.PopClip()
		}
		spans, nextState := highlightGoLineInto(c.syntaxScratch, lineText, state)
		state = nextState
		textX := bounds.MinX + editorGutterWidth - c.scrollX
		surface.PushClipRect(textClip)
		c.drawHighlightedLine(surface, textX, y+graphics.Scalar(c.baseline), lineText, spans)
		surface.PopClip()
		c.syntaxScratch = spans[:0]
		number := decimal(line + 1)
		numberX := bounds.MinX + editorGutterWidth - 8 - graphics.Scalar(len(number))*c.characterWidth
		surface.DrawText(c.Font, graphics.Point{X: numberX, Y: y + graphics.Scalar(c.baseline)}, number, lineNumberColor)
	}
	if c.Focused() && selectionStart == selectionEnd && caretLine >= firstLine && caretLine < end {
		x := bounds.MinX + editorGutterWidth + graphics.Scalar(caretColumn)*c.characterWidth - c.scrollX
		y := bounds.MinY + graphics.Scalar(caretLine*c.lineHeight) - c.scrollY
		surface.PushClipRect(textClip)
		surface.FillRect(graphics.R(x, y+1, 1, graphics.Scalar(c.lineHeight-2)), textColor)
		surface.PopClip()
	}
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, 1, bounds.Height()), borderColor)
}

func (c *EditorControl) drawHighlightedLine(surface *graphics.Surface, x, baseline graphics.Scalar, line string, spans []syntaxSpan) {
	for i := 0; i < len(spans); i++ {
		span := spans[i]
		text := line[span.start:span.end]
		surface.DrawText(c.Font, graphics.Point{X: x, Y: baseline}, text, syntaxColor(span.kind))
		x += graphics.MeasureText(c.Font, text).Width
	}
}

func syntaxColor(kind syntaxKind) graphics.Color {
	if kind == syntaxKeyword {
		return keywordColor
	}
	if kind == syntaxBuiltin {
		return builtinColor
	}
	if kind == syntaxString {
		return stringColor
	}
	if kind == syntaxComment {
		return commentColor
	}
	if kind == syntaxNumber {
		return numberColor
	}
	return textColor
}

func (c *EditorControl) pointerDown(x, y graphics.Scalar) {
	c.dragging = true
	c.placeCaret(x, y, false)
}

func (c *EditorControl) pointerMove(x, y graphics.Scalar) {
	if c.dragging {
		c.placeCaret(x, y, true)
	}
}

func (c *EditorControl) pointerUp(x, y graphics.Scalar) { c.dragging = false }

func (c *EditorControl) pointerWheel(x, y graphics.Scalar) {
	oldX, oldY := c.scrollX, c.scrollY
	c.scrollX -= x
	c.scrollY -= y
	c.clampScroll()
	if oldX != c.scrollX || oldY != c.scrollY {
		c.Invalidate()
	}
}

func (c *EditorControl) placeCaret(x, y graphics.Scalar, extend bool) {
	oldLine, _ := c.Document.Position(c.Document.Caret)
	line := int(c.scrollY+y) / c.lineHeight
	column := int((c.scrollX + x - graphics.Scalar(editorGutterWidth)) / c.characterWidth)
	if column < 0 {
		column = 0
	}
	c.Document.SetSelection(c.Document.Anchor, c.Document.VisualOffset(line, column))
	if !extend {
		c.Document.SetSelection(c.Document.Caret, c.Document.Caret)
	}
	c.ensureCaretVisible()
	newLine, _ := c.Document.Position(c.Document.Caret)
	c.invalidateEditorLine(oldLine)
	c.invalidateEditorLine(newLine)
}

func (c *EditorControl) textInput(text string) {
	var filtered []byte
	for i := 0; i < len(text); i++ {
		value := text[i]
		if value == '\r' {
			filtered = append(filtered, '\n')
		} else if value == '\n' || value == '\t' || value >= 32 {
			filtered = append(filtered, value)
		}
	}
	if len(filtered) == 0 {
		return
	}
	startLine, _ := c.Document.Position(c.Document.Caret)
	oldLines := c.Document.LineCount()
	c.Document.Insert(string(filtered))
	c.afterEdit(startLine, oldLines)
}

func (c *EditorControl) keyDown(event graphics.Event) {
	extend := event.Modifiers&graphics.ModifierShift != 0
	primary := event.Modifiers&(graphics.ModifierControl|graphics.ModifierCommand) != 0
	oldLine, _ := c.Document.Position(c.Document.Caret)
	oldLines := c.Document.LineCount()
	changed := false
	if primary && event.Key == graphics.KeyA {
		c.Document.SelectAll()
	} else if primary && event.Key == graphics.KeyC {
		graphics.SetClipboardText(c.Document.SelectedText())
		return
	} else if primary && event.Key == graphics.KeyX {
		graphics.SetClipboardText(c.Document.SelectedText())
		if c.Document.SelectedText() != "" {
			c.Document.Delete()
			changed = true
		}
	} else if primary && event.Key == graphics.KeyV {
		if text, ok := graphics.ClipboardText(); ok {
			c.Document.Paste(text)
			changed = true
		}
	} else if primary && event.Key == graphics.KeyZ {
		if extend {
			changed = c.Document.Redo()
		} else {
			changed = c.Document.Undo()
		}
	} else if primary && event.Key == graphics.KeyY {
		changed = c.Document.Redo()
	} else if primary && event.Key == graphics.KeyS {
		if c.Save != nil {
			c.Save()
		}
		return
	} else if event.Key == graphics.KeyLeft {
		c.Document.MoveCharacter(-1, extend)
	} else if event.Key == graphics.KeyRight {
		c.Document.MoveCharacter(1, extend)
	} else if event.Key == graphics.KeyUp {
		c.Document.MoveLine(-1, extend)
	} else if event.Key == graphics.KeyDown {
		c.Document.MoveLine(1, extend)
	} else if event.Key == graphics.KeyHome {
		c.Document.MoveHome(extend)
	} else if event.Key == graphics.KeyEnd {
		c.Document.MoveEnd(extend)
	} else if event.Key == graphics.KeyBackspace {
		c.Document.Backspace()
		changed = true
	} else if event.Key == graphics.KeyDelete {
		c.Document.Delete()
		changed = true
	} else {
		return
	}
	if changed {
		c.afterEdit(oldLine, oldLines)
		return
	}
	c.ensureCaretVisible()
	newLine, _ := c.Document.Position(c.Document.Caret)
	c.invalidateEditorLine(oldLine)
	c.invalidateEditorLine(newLine)
}

func (c *EditorControl) afterEdit(startLine, oldLines int) {
	c.rebuildSyntaxStates(startLine)
	c.ensureCaretVisible()
	if oldLines != c.Document.LineCount() {
		bounds := c.Bounds()
		y := bounds.MinY + graphics.Scalar(startLine*c.lineHeight) - c.scrollY
		if y < bounds.MinY {
			y = bounds.MinY
		}
		c.Form().Invalidate(graphics.R(bounds.MinX, y, bounds.Width(), bounds.MaxY-y))
	} else {
		c.invalidateEditorLine(startLine)
		line, _ := c.Document.Position(c.Document.Caret)
		c.invalidateEditorLine(line)
	}
}

func (c *EditorControl) rebuildSyntaxStates(startLine int) {
	lineCount := c.Document.LineCount()
	changedEndLine, _ := c.Document.Position(c.Document.Caret)
	sameLineCount := len(c.syntaxStates) == lineCount+1
	if startLine < 0 {
		startLine = 0
	}
	if startLine > lineCount {
		startLine = lineCount
	}
	if !sameLineCount {
		old := c.syntaxStates
		c.syntaxStates = make([]goSyntaxState, lineCount+1)
		prefix := startLine + 1
		if prefix > len(old) {
			prefix = len(old)
		}
		copy(c.syntaxStates[:prefix], old[:prefix])
	}
	if startLine == 0 {
		c.syntaxStates[0] = goSyntaxNormal
	}
	state := c.syntaxStates[startLine]
	for line := startLine; line < lineCount; line++ {
		oldNext := c.syntaxStates[line+1]
		spans, next := highlightGoLineInto(c.syntaxScratch, c.Document.LineText(line), state)
		c.syntaxScratch = spans[:0]
		c.syntaxStates[line+1] = next
		state = next
		if sameLineCount && line >= changedEndLine && next == oldNext {
			break
		}
	}
}

func (c *EditorControl) ensureCaretVisible() {
	line, column := c.Document.VisualPosition(c.Document.Caret)
	visibleLines, visibleColumns := c.VisibleGrid()
	oldX, oldY := c.scrollX, c.scrollY
	verticalMargin := editorVerticalScrollMargin
	if visibleLines <= verticalMargin*2 {
		verticalMargin = visibleLines / 3
	}
	caretTop := graphics.Scalar(line * c.lineHeight)
	caretBottom := caretTop + graphics.Scalar(c.lineHeight)
	marginY := graphics.Scalar(verticalMargin * c.lineHeight)
	if caretTop < c.scrollY+marginY {
		c.scrollY = caretTop - marginY
	}
	viewportHeight := c.Bounds().Height()
	if visibleLines > 0 && caretBottom > c.scrollY+viewportHeight-marginY {
		c.scrollY = caretBottom - viewportHeight + marginY
	}
	horizontalMargin := editorHorizontalScrollMargin
	if visibleColumns <= horizontalMargin*2 {
		horizontalMargin = visibleColumns / 3
	}
	caretX := graphics.Scalar(column) * c.characterWidth
	marginX := graphics.Scalar(horizontalMargin) * c.characterWidth
	if caretX < c.scrollX+marginX {
		c.scrollX = caretX - marginX
	}
	viewportWidth := c.Bounds().Width() - editorGutterWidth
	if visibleColumns > 0 && caretX+c.characterWidth > c.scrollX+viewportWidth-marginX {
		c.scrollX = caretX + c.characterWidth - viewportWidth + marginX
	}
	c.clampScroll()
	if oldX != c.scrollX || oldY != c.scrollY {
		c.Invalidate()
	}
}

func (c *EditorControl) clampScroll() {
	maximum := graphics.Scalar(c.Document.LineCount()*c.lineHeight) - c.Bounds().Height()
	if maximum < 0.0 {
		maximum = 0.0
	}
	if c.scrollY < 0.0 {
		c.scrollY = 0.0
	}
	if c.scrollY > maximum {
		c.scrollY = maximum
	}
	if c.scrollX < 0.0 {
		c.scrollX = 0.0
	}
}

func (c *EditorControl) invalidateEditorLine(line int) {
	if c.Form() == nil {
		return
	}
	bounds := c.Bounds()
	y := bounds.MinY + graphics.Scalar(line*c.lineHeight) - c.scrollY
	if y < bounds.MaxY && y+graphics.Scalar(c.lineHeight) > bounds.MinY {
		c.Form().Invalidate(graphics.R(bounds.MinX, y, bounds.Width(), graphics.Scalar(c.lineHeight)))
	}
}

func decimal(value int) string {
	if value == 0 {
		return "0"
	}
	var digits []byte
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	out := make([]byte, len(digits))
	for i := 0; i < len(digits); i++ {
		out[i] = digits[len(digits)-i-1]
	}
	return string(out)
}
