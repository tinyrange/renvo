package ide

import (
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

type editorThemePalette struct {
	background    graphics.Color
	gutter        graphics.Color
	popup         graphics.Color
	border        graphics.Color
	selection     graphics.Color
	selectionText graphics.Color
	text          graphics.Color
	lineNumber    graphics.Color
	currentLine   graphics.Color
	keyword       graphics.Color
	builtin       graphics.Color
	stringLiteral graphics.Color
	comment       graphics.Color
	number        graphics.Color
	error         graphics.Color
	warning       graphics.Color
}

func editorPalette(theme forms.Theme) editorThemePalette {
	palette := editorThemePalette{
		background:    theme.Field,
		gutter:        theme.Surface,
		popup:         theme.SurfaceRaised,
		border:        theme.Border,
		selection:     theme.Selection,
		selectionText: theme.Text,
		text:          theme.Text,
		lineNumber:    theme.MutedText,
		currentLine:   theme.SurfaceRaised,
		keyword:       graphics.RGBA(0, 82, 180, 255),
		builtin:       graphics.RGBA(0, 118, 118, 255),
		stringLiteral: graphics.RGBA(156, 36, 33, 255),
		comment:       graphics.RGBA(38, 128, 62, 255),
		number:        graphics.RGBA(108, 62, 176, 255),
		error:         graphics.RGBA(211, 47, 47, 255),
		warning:       graphics.RGBA(183, 121, 0, 255),
	}
	if int(theme.Window.R)+int(theme.Window.G)+int(theme.Window.B) < 384 {
		palette.keyword = graphics.RGBA(86, 156, 214, 255)
		palette.builtin = graphics.RGBA(78, 201, 176, 255)
		palette.stringLiteral = graphics.RGBA(206, 145, 120, 255)
		palette.comment = graphics.RGBA(106, 153, 85, 255)
		palette.number = graphics.RGBA(181, 206, 168, 255)
		palette.error = graphics.RGBA(244, 112, 104, 255)
		palette.warning = graphics.RGBA(220, 177, 90, 255)
	}
	return palette
}

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
	control.SetAccessibilityRole(forms.AccessibilityRoleTree)
	control.SetAccessibilityName("Project explorer")
	control.AccessibilityChildren = control.accessibilityChildren
	control.AccessibilityPerform = control.accessibilityPerform
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

func (c *ExplorerControl) SetModel(model *Explorer) {
	if c == nil || model == nil || c.Model == model {
		return
	}
	c.Model = model
	c.scrollY = 0
	c.AccessibilityChildrenChanged()
	c.Invalidate()
}

func (c *ExplorerControl) RowHeight() int { return c.rowHeight }

func (c *ExplorerControl) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	palette := editorPalette(c.Theme())
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
		color := palette.text
		if i == c.Model.SelectedIndex() {
			surface.FillRect(rowRect, palette.selection)
			color = palette.selectionText
		}
		node := rows[i].Node
		x := bounds.MinX + 16 + graphics.Scalar(rows[i].Depth*18)
		if node.Directory {
			drawExplorerChevron(surface, x, y+graphics.Scalar(c.rowHeight/2-3), node.Expanded, color)
			drawExplorerFolder(surface, x+15, y+graphics.Scalar(c.rowHeight/2-7), palette)
		} else {
			drawExplorerFile(surface, x+15, y+graphics.Scalar(c.rowHeight/2-8), palette)
		}
		surface.DrawText(c.Font, graphics.Point{X: x + 36, Y: y + graphics.Scalar(c.baseline)}, node.Name, color)
	}
	surface.FillRect(graphics.R(bounds.MaxX-1, bounds.MinY, 1, bounds.Height()), palette.border)
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

func drawExplorerFolder(surface *graphics.Surface, x, y graphics.Scalar, palette editorThemePalette) {
	surface.FillRect(graphics.R(x+1, y, 7, 3), palette.currentLine)
	surface.FillRect(graphics.R(x, y+3, 16, 11), palette.currentLine)
	surface.StrokeRect(graphics.R(x, y+3, 16, 11), 1, palette.lineNumber)
}

func drawExplorerFile(surface *graphics.Surface, x, y graphics.Scalar, palette editorThemePalette) {
	surface.StrokeRect(graphics.R(x+2, y, 12, 15), 1, palette.lineNumber)
}

func (c *ExplorerControl) pointerDown(x, y graphics.Scalar) {
	index := int(c.scrollY+y) / c.rowHeight
	rows := c.Model.Rows()
	if index < 0 || index >= len(rows) {
		return
	}
	old := c.Model.SelectedIndex()
	c.Model.Select(index)
	c.AccessibilityChildrenChanged()
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
		c.AccessibilityChildrenChanged()
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
	c.AccessibilityChildrenChanged()
	if oldRows != len(c.Model.Rows()) {
		c.Invalidate()
	} else {
		c.invalidateRow(old)
		c.invalidateRow(c.Model.SelectedIndex())
	}
}

func (c *ExplorerControl) accessibilityChildren() []forms.AccessibilityNode {
	if c == nil || c.Model == nil {
		return nil
	}
	rows := c.Model.Rows()
	nodes := make([]forms.AccessibilityNode, 0, len(rows))
	bounds := c.Bounds()
	for i := 0; i < len(rows); i++ {
		y := bounds.MinY + graphics.Scalar(i*c.rowHeight) - c.scrollY
		nodes = append(nodes, forms.AccessibilityNode{
			ID:         c.AccessibilityID() + "-row-" + decimal(i+1),
			Role:       forms.AccessibilityRoleTreeItem,
			Name:       rows[i].Node.Name,
			Bounds:     graphics.R(bounds.MinX, y, bounds.Width(), graphics.Scalar(c.rowHeight)),
			Actions:    forms.AccessibilitySupportsInvoke,
			Selectable: true,
			Selected:   i == c.Model.SelectedIndex(),
		})
	}
	return nodes
}

func (c *ExplorerControl) accessibilityPerform(id string, action forms.AccessibilityAction, value string) bool {
	index, ok := accessibilityIndex(id, c.AccessibilityID()+"-row-")
	if !ok || c.Model == nil || index < 0 || index >= len(c.Model.Rows()) {
		return false
	}
	c.Model.Select(index)
	c.ensureSelectionVisible()
	c.Invalidate()
	c.AccessibilityChildrenStateChanged()
	if action == forms.AccessibilityActionInvoke {
		c.activate()
		return true
	}
	return action == forms.AccessibilityActionFocus
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
	Document        *Document
	Font            *graphics.Font
	Save            func()
	Changed         func()
	Complete        func(source []byte, caret int) []Completion
	Signature       func(source []byte, caret int, help *SignatureHelp)
	scrollY         graphics.Scalar
	scrollX         graphics.Scalar
	dragging        bool
	lineHeight      int
	characterWidth  graphics.Scalar
	baseline        int
	syntaxScratch   []syntaxSpan
	syntaxStates    []goSyntaxState
	completions     []Completion
	completionAll   []Completion
	completionWork  []Completion
	completionAt    int
	completionPick  int
	completionFirst int
	signature       SignatureHelp
	signatureOpen   int
	diagnostics     []Diagnostic
}

type Completion struct {
	Text       string
	Detail     string
	Kind       int
	Signature  string
	Parameters []SignatureParameter
}

type SignatureParameter struct {
	Name string
	Type string
}

type SignatureHelp struct {
	Ok              bool
	Label           string
	Parameters      []SignatureParameter
	ActiveParameter int
}

type Diagnostic struct {
	Start   int
	End     int
	Message string
	Error   bool
}

func NewEditorControl(document *Document) *EditorControl {
	control := &EditorControl{Document: document, signatureOpen: -1}
	control.Control = *forms.NewControl()
	control.SetThemeRole(forms.ThemeField)
	control.SetAcceptsTab(true)
	control.SetFont(graphics.NewBuiltinFont(2))
	control.SetAccessibilityRole(forms.AccessibilityRoleTextBox)
	control.SetAccessibilityName("Code editor")
	control.SetAccessibilityMultiline(true)
	control.AccessibilityValue = control.accessibilityValue
	control.AccessibilitySetValue = control.accessibilitySetValue
	control.AccessibilitySelectionStart = control.accessibilitySelectionStart
	control.AccessibilitySelectionEnd = control.accessibilitySelectionEnd
	control.AccessibilitySetSelection = control.accessibilitySetSelection
	control.AccessibilityChildren = control.accessibilityChildren
	control.AccessibilityPerform = control.accessibilityPerform
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
	c.diagnostics = nil
	c.closeCompletion()
	c.closeSignature()
	c.rebuildSyntaxStates(0)
	c.AccessibilityChanged()
	c.AccessibilityChildrenChanged()
	c.Invalidate()
}

func (c *EditorControl) SetDiagnostics(diagnostics []Diagnostic) {
	if c == nil {
		return
	}
	c.invalidateDiagnostics(c.diagnostics)
	copyOfDiagnostics := make([]Diagnostic, len(diagnostics))
	copy(copyOfDiagnostics, diagnostics)
	c.diagnostics = copyOfDiagnostics
	c.AccessibilityChildrenChanged()
	c.invalidateDiagnostics(c.diagnostics)
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
	palette := editorPalette(c.Theme())
	c.clampScroll()
	surface.FillRect(bounds, c.Background())
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, editorGutterWidth, bounds.Height()), palette.gutter)
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
			surface.FillRect(lineRect, palette.currentLine)
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
			surface.FillRect(graphics.R(x, y, width, graphics.Scalar(c.lineHeight)), palette.selection)
			surface.PopClip()
		}
		spans, nextState := highlightGoLineInto(c.syntaxScratch, lineText, state)
		state = nextState
		textX := bounds.MinX + editorGutterWidth - c.scrollX
		surface.PushClipRect(textClip)
		c.drawHighlightedLine(surface, textX, y+graphics.Scalar(c.baseline), lineText, spans, palette)
		c.drawLineDiagnostics(surface, line, lineStart, lineEnd, y, textX, palette)
		surface.PopClip()
		c.syntaxScratch = spans[:0]
		number := decimal(line + 1)
		numberX := bounds.MinX + editorGutterWidth - 8 - graphics.Scalar(len(number))*c.characterWidth
		surface.DrawText(c.Font, graphics.Point{X: numberX, Y: y + graphics.Scalar(c.baseline)}, number, palette.lineNumber)
	}
	if c.Focused() && selectionStart == selectionEnd && caretLine >= firstLine && caretLine < end {
		x := bounds.MinX + editorGutterWidth + graphics.Scalar(caretColumn)*c.characterWidth - c.scrollX
		y := bounds.MinY + graphics.Scalar(caretLine*c.lineHeight) - c.scrollY
		surface.PushClipRect(textClip)
		surface.FillRect(graphics.R(x, y+1, 1, graphics.Scalar(c.lineHeight-2)), palette.text)
		surface.PopClip()
	}
	c.paintCompletion(surface)
	c.paintSignature(surface)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, 1, bounds.Height()), palette.border)
}

func (c *EditorControl) drawLineDiagnostics(surface *graphics.Surface, line, lineStart, lineEnd int, y, textX graphics.Scalar, palette editorThemePalette) {
	for i := 0; i < len(c.diagnostics); i++ {
		diagnostic := c.diagnostics[i]
		start, end := diagnostic.Start, diagnostic.End
		if end <= start {
			end = start + 1
		}
		if start > lineEnd || end < lineStart {
			continue
		}
		if start < lineStart {
			start = lineStart
		}
		if end > lineEnd {
			end = lineEnd
		}
		_, startColumn := c.Document.VisualPosition(start)
		_, endColumn := c.Document.VisualPosition(end)
		if endColumn <= startColumn {
			endColumn = startColumn + 1
		}
		x := textX + graphics.Scalar(startColumn)*c.characterWidth
		maxX := textX + graphics.Scalar(endColumn)*c.characterWidth
		underlineY := y + graphics.Scalar(c.lineHeight-2)
		color := palette.error
		if !diagnostic.Error {
			color = palette.warning
		}
		up := false
		for x < maxX {
			next := x + 2
			if next > maxX {
				next = maxX
			}
			fromY, toY := underlineY, underlineY-1
			if up {
				fromY, toY = underlineY-1, underlineY
			}
			surface.DrawLine(graphics.Point{X: x, Y: fromY}, graphics.Point{X: next, Y: toY}, 1, color)
			x = next
			up = !up
		}
	}
}

func (c *EditorControl) invalidateDiagnostics(diagnostics []Diagnostic) {
	if c == nil || c.Document == nil || c.Form() == nil {
		return
	}
	for i := 0; i < len(diagnostics); i++ {
		startLine, _ := c.Document.Position(diagnostics[i].Start)
		endLine, _ := c.Document.Position(diagnostics[i].End)
		for line := startLine; line <= endLine; line++ {
			c.invalidateEditorLine(line)
		}
	}
}

func (c *EditorControl) drawHighlightedLine(surface *graphics.Surface, x, baseline graphics.Scalar, line string, spans []syntaxSpan, palette editorThemePalette) {
	for i := 0; i < len(spans); i++ {
		span := spans[i]
		text := line[span.start:span.end]
		surface.DrawText(c.Font, graphics.Point{X: x, Y: baseline}, text, syntaxColor(span.kind, palette))
		x += graphics.MeasureText(c.Font, text).Width
	}
}

func syntaxColor(kind syntaxKind, palette editorThemePalette) graphics.Color {
	if kind == syntaxKeyword {
		return palette.keyword
	}
	if kind == syntaxBuiltin {
		return palette.builtin
	}
	if kind == syntaxString {
		return palette.stringLiteral
	}
	if kind == syntaxComment {
		return palette.comment
	}
	if kind == syntaxNumber {
		return palette.number
	}
	return palette.text
}

func (c *EditorControl) pointerDown(x, y graphics.Scalar) {
	c.closeCompletion()
	c.closeSignature()
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
	c.AccessibilitySelectionChanged()
}

func (c *EditorControl) textInput(text string) {
	var filtered []byte
	for i := 0; i < len(text); i++ {
		value := text[i]
		if value == '\r' {
			filtered = append(filtered, '\n')
		} else if value == '\n' || value >= 32 {
			filtered = append(filtered, value)
		}
	}
	if len(filtered) == 0 {
		return
	}
	hadCompletion := len(c.completions) > 0
	if hadCompletion {
		c.invalidateCompletion()
	}
	editStart, _ := c.Document.Selection()
	startLine, _ := c.Document.Position(editStart)
	oldLines := c.Document.LineCount()
	c.Document.Insert(string(filtered))
	c.afterEdit(startLine, oldLines)
	if hadCompletion && completionIdentifierText(filtered) {
		c.refilterCompletion(false)
	} else {
		c.closeCompletion()
		if len(filtered) == 1 && filtered[0] == '.' {
			c.refreshCompletion(false, false)
		}
	}
	if signatureTriggerText(filtered) {
		c.refreshSignature()
	} else if c.signature.Ok {
		c.maintainSignature()
	}
}

func (c *EditorControl) keyDown(event graphics.Event) {
	extend := event.Modifiers&graphics.ModifierShift != 0
	control := event.Modifiers&graphics.ModifierControl != 0
	command := event.Modifiers&graphics.ModifierCommand != 0
	primary := control || command
	oldLine, _ := c.Document.Position(c.Document.Caret)
	oldLines := c.Document.LineCount()
	changed := false
	if len(c.completions) > 0 {
		if event.Key == graphics.KeyUp {
			c.completionPick--
			if c.completionPick < 0 {
				c.completionPick = len(c.completions) - 1
			}
			c.ensureCompletionPickVisible()
			c.invalidateCompletion()
			c.AccessibilityChildrenStateChanged()
			return
		}
		if event.Key == graphics.KeyDown {
			c.completionPick++
			if c.completionPick >= len(c.completions) {
				c.completionPick = 0
			}
			c.ensureCompletionPickVisible()
			c.invalidateCompletion()
			c.AccessibilityChildrenStateChanged()
			return
		}
		if event.Key == graphics.KeyPageUp {
			c.completionPick -= 8
			if c.completionPick < 0 {
				c.completionPick = 0
			}
			c.ensureCompletionPickVisible()
			c.invalidateCompletion()
			c.AccessibilityChildrenStateChanged()
			return
		}
		if event.Key == graphics.KeyPageDown {
			c.completionPick += 8
			if c.completionPick >= len(c.completions) {
				c.completionPick = len(c.completions) - 1
			}
			c.ensureCompletionPickVisible()
			c.invalidateCompletion()
			c.AccessibilityChildrenStateChanged()
			return
		}
		if event.Key == graphics.KeyTab || event.Key == graphics.KeyEnter {
			c.acceptCompletion()
			return
		}
		if event.Key == graphics.KeyEscape {
			c.closeCompletion()
			c.closeSignature()
			return
		}
	}
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
	} else if extend && event.Key == graphics.KeySpace && (control || command) {
		c.refreshSignature()
		return
	} else if control && event.Key == graphics.KeySpace || command && event.Key == graphics.KeyI {
		if c.refreshCompletion(false, true) {
			return
		}
	} else if event.Key == graphics.KeyTab {
		if c.refreshCompletion(true, false) {
			return
		}
		c.Document.Insert("\t")
		changed = true
	} else if event.Key == graphics.KeyEscape {
		c.closeCompletion()
		c.closeSignature()
		return
	} else if event.Key == graphics.KeyLeft {
		c.closeCompletion()
		c.Document.MoveCharacter(-1, extend)
	} else if event.Key == graphics.KeyRight {
		c.closeCompletion()
		c.Document.MoveCharacter(1, extend)
	} else if event.Key == graphics.KeyUp {
		c.Document.MoveLine(-1, extend)
	} else if event.Key == graphics.KeyDown {
		c.Document.MoveLine(1, extend)
	} else if event.Key == graphics.KeyHome {
		c.closeCompletion()
		c.Document.MoveHome(extend)
	} else if event.Key == graphics.KeyEnd {
		c.closeCompletion()
		c.Document.MoveEnd(extend)
	} else if event.Key == graphics.KeyBackspace {
		if !c.Document.Backspace() {
			return
		}
		changed = true
	} else if event.Key == graphics.KeyDelete {
		if !c.Document.Delete() {
			return
		}
		changed = true
	} else {
		return
	}
	if changed {
		startLine, _ := c.Document.Position(c.Document.Caret)
		if oldLine < startLine {
			startLine = oldLine
		}
		c.afterEdit(startLine, oldLines)
		if len(c.completions) > 0 {
			c.refilterCompletion(false)
		}
		if c.signature.Ok {
			c.refreshSignature()
		}
		return
	}
	c.ensureCaretVisible()
	newLine, _ := c.Document.Position(c.Document.Caret)
	c.invalidateEditorLine(oldLine)
	c.invalidateEditorLine(newLine)
	c.AccessibilitySelectionChanged()
	if c.signature.Ok {
		c.refreshSignature()
	}
}

func (c *EditorControl) refreshCompletion(acceptSingle, explicit bool) bool {
	if len(c.completions) > 0 {
		c.invalidateCompletion()
	}
	if c.Complete == nil || c.Document == nil {
		c.closeCompletion()
		return false
	}
	if !explicit && (c.Document.Caret <= 0 || completionWhitespace(c.Document.text[c.Document.Caret-1])) {
		c.closeCompletion()
		return false
	}
	start := completionWordStart(c.Document.text, c.Document.Caret)
	items := c.Complete(c.Document.text, c.Document.Caret)
	if len(items) == 0 {
		c.closeCompletion()
		return false
	}
	c.completionAt = start
	c.completionAll = items
	c.completionWork = make([]Completion, 0, len(items))
	c.filterCompletionItems()
	if len(c.completions) == 0 {
		c.closeCompletion()
		return false
	}
	if acceptSingle && len(c.completions) == 1 {
		c.acceptCompletion()
		return true
	}
	c.AccessibilityChildrenChanged()
	c.invalidateCompletion()
	return true
}

func (c *EditorControl) refilterCompletion(acceptSingle bool) bool {
	if len(c.completionAll) == 0 || c.Document == nil || c.Document.Caret < c.completionAt {
		c.closeCompletion()
		return false
	}
	if len(c.completions) > 0 {
		c.invalidateCompletion()
	}
	c.filterCompletionItems()
	if len(c.completions) == 0 {
		c.closeCompletion()
		return false
	}
	if acceptSingle && len(c.completions) == 1 {
		c.acceptCompletion()
		return true
	}
	c.AccessibilityChildrenChanged()
	c.invalidateCompletion()
	return true
}

func (c *EditorControl) filterCompletionItems() {
	pattern := c.Document.text[c.completionAt:c.Document.Caret]
	c.completionWork = c.completionWork[:0]
	for pass := 0; pass < 2; pass++ {
		for i := 0; i < len(c.completionAll); i++ {
			name := c.completionAll[i].Text
			if completionTextEqual(name, pattern) {
				continue
			}
			prefix := completionTextPrefix(name, pattern)
			if pass == 0 && prefix || pass == 1 && !prefix && completionTextFuzzy(name, pattern) {
				c.completionWork = append(c.completionWork, c.completionAll[i])
			}
		}
	}
	c.completions = c.completionWork
	if c.completionPick < 0 || c.completionPick >= len(c.completions) {
		c.completionPick = 0
	}
	c.ensureCompletionPickVisibleFor(len(c.completions))
}

func (c *EditorControl) ensureCompletionPickVisible() {
	c.ensureCompletionPickVisibleFor(len(c.completions))
}

func (c *EditorControl) ensureCompletionPickVisibleFor(count int) {
	const visibleRows = 8
	if count <= visibleRows {
		c.completionFirst = 0
		return
	}
	if c.completionPick < c.completionFirst {
		c.completionFirst = c.completionPick
	}
	if c.completionPick >= c.completionFirst+visibleRows {
		c.completionFirst = c.completionPick - visibleRows + 1
	}
	maximum := count - visibleRows
	if c.completionFirst > maximum {
		c.completionFirst = maximum
	}
	if c.completionFirst < 0 {
		c.completionFirst = 0
	}
}

func (c *EditorControl) refreshSignature() {
	if c.Signature == nil || c.Document == nil {
		c.closeSignature()
		return
	}
	old := c.signature
	if old.Ok {
		c.invalidateSignature()
	}
	var next SignatureHelp
	c.Signature(c.Document.text, c.Document.Caret, &next)
	open := signatureOpenBefore(c.Document.text, c.Document.Caret)
	if next.Ok && open >= 0 {
		active, ok := signatureContext(c.Document.text, c.Document.Caret, open)
		if !ok {
			c.closeSignature()
			return
		}
		next.ActiveParameter = active
		c.signature = next
		c.signatureOpen = open
		c.AccessibilityChildrenChanged()
		c.invalidateSignature()
		return
	}
	if old.Ok {
		c.signature = old
		c.maintainSignature()
		return
	}
	c.signature = SignatureHelp{}
	c.signatureOpen = -1
}

func (c *EditorControl) maintainSignature() {
	if !c.signature.Ok || c.Document == nil {
		return
	}
	active, ok := signatureContext(c.Document.text, c.Document.Caret, c.signatureOpen)
	if !ok {
		c.closeSignature()
		return
	}
	if active != c.signature.ActiveParameter {
		c.invalidateSignature()
		c.signature.ActiveParameter = active
		c.invalidateSignature()
	}
}

func (c *EditorControl) closeSignature() {
	if !c.signature.Ok {
		c.signatureOpen = -1
		return
	}
	c.invalidateSignature()
	c.signature = SignatureHelp{}
	c.signatureOpen = -1
	c.AccessibilityChildrenChanged()
}

func (c *EditorControl) acceptCompletion() {
	if len(c.completions) == 0 || c.completionPick < 0 || c.completionPick >= len(c.completions) {
		return
	}
	c.invalidateCompletion()
	startLine, _ := c.Document.Position(c.completionAt)
	oldLines := c.Document.LineCount()
	text := c.completions[c.completionPick].Text
	c.Document.SetSelection(c.completionAt, c.Document.Caret)
	c.Document.Insert(text)
	c.completions = nil
	c.completionAll = nil
	c.completionWork = nil
	c.completionPick = 0
	c.completionFirst = 0
	c.AccessibilityChildrenChanged()
	c.afterEdit(startLine, oldLines)
}

func (c *EditorControl) closeCompletion() {
	if len(c.completions) == 0 && len(c.completionAll) == 0 {
		return
	}
	c.invalidateCompletion()
	c.completions = nil
	c.completionAll = nil
	c.completionWork = nil
	c.completionPick = 0
	c.completionFirst = 0
	c.AccessibilityChildrenChanged()
}

func (c *EditorControl) paintCompletion(surface *graphics.Surface) {
	if len(c.completions) == 0 || c.Font == nil {
		return
	}
	bounds := c.completionBounds()
	palette := editorPalette(c.Theme())
	surface.FillRect(bounds, palette.popup)
	surface.StrokeRect(bounds, 1, palette.border)
	rowHeight := graphics.Scalar(c.lineHeight + 4)
	end := c.completionFirst + 8
	if end > len(c.completions) {
		end = len(c.completions)
	}
	for i := c.completionFirst; i < end; i++ {
		rowIndex := i - c.completionFirst
		row := graphics.R(bounds.MinX+1, bounds.MinY+1+graphics.Scalar(rowIndex)*rowHeight, bounds.Width()-2, rowHeight)
		if i == c.completionPick {
			surface.FillRect(row, palette.selection)
		}
		surface.DrawText(c.Font, graphics.Point{X: row.MinX + 8, Y: row.MinY + graphics.Scalar(c.baseline+2)}, c.completions[i].Text, palette.text)
		detailWidth := graphics.MeasureText(c.Font, c.completions[i].Detail).Width
		surface.DrawText(c.Font, graphics.Point{X: row.MaxX - detailWidth - 8, Y: row.MinY + graphics.Scalar(c.baseline+2)}, c.completions[i].Detail, palette.lineNumber)
	}
}

func (c *EditorControl) paintSignature(surface *graphics.Surface) {
	if !c.signature.Ok || c.Font == nil || c.signature.Label == "" {
		return
	}
	bounds := c.signatureBounds()
	palette := editorPalette(c.Theme())
	surface.FillRect(bounds, palette.popup)
	surface.StrokeRect(bounds, 1, palette.border)
	surface.DrawText(c.Font, graphics.Point{X: bounds.MinX + 8, Y: bounds.MinY + graphics.Scalar(c.baseline+3)}, c.signature.Label, palette.text)
	active := c.signature.ActiveParameter
	if active >= 0 && active < len(c.signature.Parameters) {
		parameter := c.signature.Parameters[active]
		label := "argument " + decimal(active+1) + ": "
		if parameter.Name != "" {
			label += parameter.Name + " "
		}
		label += parameter.Type
		surface.DrawText(c.Font, graphics.Point{X: bounds.MinX + 8, Y: bounds.MinY + graphics.Scalar(c.lineHeight+c.baseline+1)}, label, palette.lineNumber)
	}
}

func (c *EditorControl) signatureBounds() graphics.Rect {
	line, column := c.Document.VisualPosition(c.Document.Caret)
	width := graphics.Scalar(480)
	if width > c.Bounds().Width()-editorGutterWidth-8 {
		width = c.Bounds().Width() - editorGutterWidth - 8
	}
	height := graphics.Scalar(c.lineHeight*2 + 4)
	x := c.Bounds().MinX + editorGutterWidth + graphics.Scalar(column)*c.characterWidth - c.scrollX
	y := c.Bounds().MinY + graphics.Scalar((line+1)*c.lineHeight) - c.scrollY
	if len(c.completions) > 0 {
		y = c.completionBounds().MinY - height - 2
	}
	if x+width > c.Bounds().MaxX-4 {
		x = c.Bounds().MaxX - width - 4
	}
	if y+height > c.Bounds().MaxY-4 {
		y = c.Bounds().MinY + graphics.Scalar(line*c.lineHeight) - c.scrollY - height
	}
	return graphics.R(x, y, width, height)
}

func (c *EditorControl) invalidateSignature() {
	if c.Form() != nil && c.signature.Ok {
		c.Form().Invalidate(c.signatureBounds())
	}
}

func (c *EditorControl) completionBounds() graphics.Rect {
	line, column := c.Document.VisualPosition(c.Document.Caret)
	width := graphics.Scalar(330)
	if width > c.Bounds().Width()-editorGutterWidth-8 {
		width = c.Bounds().Width() - editorGutterWidth - 8
	}
	rows := len(c.completions)
	if rows > 8 {
		rows = 8
	}
	height := graphics.Scalar(rows*(c.lineHeight+4) + 2)
	x := c.Bounds().MinX + editorGutterWidth + graphics.Scalar(column)*c.characterWidth - c.scrollX
	y := c.Bounds().MinY + graphics.Scalar((line+1)*c.lineHeight) - c.scrollY
	if x+width > c.Bounds().MaxX-4 {
		x = c.Bounds().MaxX - width - 4
	}
	if y+height > c.Bounds().MaxY-4 {
		y = c.Bounds().MinY + graphics.Scalar(line*c.lineHeight) - c.scrollY - height
	}
	return graphics.R(x, y, width, height)
}

func (c *EditorControl) invalidateCompletion() {
	if c.Form() != nil && len(c.completions) > 0 {
		c.Form().Invalidate(c.completionBounds())
	}
}

func completionWordStart(data []byte, offset int) int {
	if offset > len(data) {
		offset = len(data)
	}
	for offset > 0 {
		value := data[offset-1]
		if value != '_' && (value < 'a' || value > 'z') && (value < 'A' || value > 'Z') && (value < '0' || value > '9') {
			break
		}
		offset--
	}
	return offset
}

func completionWhitespace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\n' || value == '\r'
}

func completionIdentifierText(text []byte) bool {
	if len(text) == 0 {
		return false
	}
	for i := 0; i < len(text); i++ {
		value := text[i]
		if value != '_' && (value < 'a' || value > 'z') && (value < 'A' || value > 'Z') && (value < '0' || value > '9') {
			return false
		}
	}
	return true
}

func completionTextEqual(name string, pattern []byte) bool {
	if len(name) != len(pattern) {
		return false
	}
	for i := 0; i < len(pattern); i++ {
		if completionLower(name[i]) != completionLower(pattern[i]) {
			return false
		}
	}
	return true
}

func completionTextPrefix(name string, pattern []byte) bool {
	if len(pattern) > len(name) {
		return false
	}
	for i := 0; i < len(pattern); i++ {
		if completionLower(name[i]) != completionLower(pattern[i]) {
			return false
		}
	}
	return true
}

func completionTextFuzzy(name string, pattern []byte) bool {
	if len(pattern) == 0 {
		return true
	}
	at := 0
	for i := 0; i < len(name) && at < len(pattern); i++ {
		if completionLower(name[i]) == completionLower(pattern[at]) {
			at++
		}
	}
	return at == len(pattern)
}

func completionLower(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	return value
}

func signatureTriggerText(text []byte) bool {
	for i := 0; i < len(text); i++ {
		if text[i] == '(' || text[i] == ',' || text[i] == ')' {
			return true
		}
	}
	return false
}

func signatureOpenBefore(data []byte, caret int) int {
	if caret > len(data) {
		caret = len(data)
	}
	stack := make([]int, 0, 8)
	state := byte(0)
	for i := 0; i < caret; i++ {
		value := data[i]
		if state == '/' {
			if value == '\n' {
				state = 0
			}
			continue
		}
		if state == '*' {
			if value == '*' && i+1 < caret && data[i+1] == '/' {
				i++
				state = 0
			}
			continue
		}
		if state == '`' {
			if value == '`' {
				state = 0
			}
			continue
		}
		if state == '"' || state == '\'' {
			if value == '\\' {
				i++
				continue
			}
			if value == state {
				state = 0
			}
			continue
		}
		if value == '/' && i+1 < caret {
			if data[i+1] == '/' {
				i++
				state = '/'
				continue
			}
			if data[i+1] == '*' {
				i++
				state = '*'
				continue
			}
		}
		if value == '"' || value == '\'' || value == '`' {
			state = value
			continue
		}
		if value == '(' {
			stack = append(stack, i)
		} else if value == ')' && len(stack) > 0 {
			stack = stack[:len(stack)-1]
		}
	}
	if len(stack) == 0 {
		return -1
	}
	return stack[len(stack)-1]
}

func signatureContext(data []byte, caret, open int) (int, bool) {
	if open < 0 || open >= caret || open >= len(data) || data[open] != '(' {
		return 0, false
	}
	if caret > len(data) {
		caret = len(data)
	}
	active := 0
	paren, bracket, brace := 0, 0, 0
	state := byte(0)
	for i := open + 1; i < caret; i++ {
		value := data[i]
		if state == '/' {
			if value == '\n' {
				state = 0
			}
			continue
		}
		if state == '*' {
			if value == '*' && i+1 < caret && data[i+1] == '/' {
				i++
				state = 0
			}
			continue
		}
		if state == '`' {
			if value == '`' {
				state = 0
			}
			continue
		}
		if state == '"' || state == '\'' {
			if value == '\\' {
				i++
				continue
			}
			if value == state {
				state = 0
			}
			continue
		}
		if value == '/' && i+1 < caret {
			if data[i+1] == '/' {
				i++
				state = '/'
				continue
			}
			if data[i+1] == '*' {
				i++
				state = '*'
				continue
			}
		}
		if value == '"' || value == '\'' || value == '`' {
			state = value
			continue
		}
		if value == '(' {
			paren++
		} else if value == ')' {
			if paren == 0 {
				return active, false
			}
			paren--
		} else if value == '[' {
			bracket++
		} else if value == ']' && bracket > 0 {
			bracket--
		} else if value == '{' {
			brace++
		} else if value == '}' && brace > 0 {
			brace--
		} else if value == ',' && paren == 0 && bracket == 0 && brace == 0 {
			active++
		}
	}
	return active, true
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
	if c.Changed != nil {
		c.Changed()
	}
	c.AccessibilityChanged()
}

func (c *EditorControl) accessibilityValue() string {
	if c == nil || c.Document == nil {
		return ""
	}
	return c.Document.Text()
}

func (c *EditorControl) accessibilitySelectionStart() int {
	if c == nil || c.Document == nil {
		return -1
	}
	start, _ := c.Document.Selection()
	return start
}

func (c *EditorControl) accessibilitySelectionEnd() int {
	if c == nil || c.Document == nil {
		return -1
	}
	_, end := c.Document.Selection()
	return end
}

func (c *EditorControl) accessibilitySetValue(value string) {
	if c == nil || c.Document == nil || c.Document.Text() == value {
		return
	}
	oldLines := c.Document.LineCount()
	c.Document.SetSelection(0, len(c.Document.text))
	c.Document.Insert(value)
	c.afterEdit(0, oldLines)
}

func (c *EditorControl) accessibilitySetSelection(start, end int) {
	if c == nil || c.Document == nil {
		return
	}
	oldLine, _ := c.Document.Position(c.Document.Caret)
	c.Document.SetSelection(start, end)
	c.ensureCaretVisible()
	newLine, _ := c.Document.Position(c.Document.Caret)
	c.invalidateEditorLine(oldLine)
	c.invalidateEditorLine(newLine)
	c.AccessibilitySelectionChanged()
}

func (c *EditorControl) accessibilityChildren() []forms.AccessibilityNode {
	if c == nil || c.Document == nil {
		return nil
	}
	nodes := make([]forms.AccessibilityNode, 0, len(c.completions)+len(c.diagnostics)+2)
	baseID := c.AccessibilityID()
	if len(c.completions) > 0 {
		listID := baseID + "-completions"
		bounds := c.completionBounds()
		nodes = append(nodes, forms.AccessibilityNode{ID: listID, Role: forms.AccessibilityRoleList, Name: "Code completions", Bounds: bounds})
		rowHeight := graphics.Scalar(c.lineHeight + 4)
		for i := 0; i < len(c.completions); i++ {
			nodes = append(nodes, forms.AccessibilityNode{
				ID:          listID + "-" + decimal(i+1),
				ParentID:    listID,
				Role:        forms.AccessibilityRoleListItem,
				Name:        c.completions[i].Text,
				Description: c.completions[i].Detail,
				Bounds:      graphics.R(bounds.MinX, bounds.MinY+graphics.Scalar(i-c.completionFirst)*rowHeight, bounds.Width(), rowHeight),
				Actions:     forms.AccessibilitySupportsInvoke,
				Selectable:  true,
				Selected:    i == c.completionPick,
			})
		}
	}
	if c.signature.Ok {
		nodes = append(nodes, forms.AccessibilityNode{ID: baseID + "-signature", Role: forms.AccessibilityRoleStatus, Name: "Signature help", Value: c.signature.Label, Bounds: c.signatureBounds()})
	}
	for i := 0; i < len(c.diagnostics); i++ {
		line, _ := c.Document.Position(c.diagnostics[i].Start)
		prefix := "Warning: "
		if c.diagnostics[i].Error {
			prefix = "Error: "
		}
		nodes = append(nodes, forms.AccessibilityNode{
			ID:     baseID + "-diagnostic-" + decimal(i+1),
			Role:   forms.AccessibilityRoleStatus,
			Name:   prefix + c.diagnostics[i].Message,
			Bounds: graphics.R(c.Bounds().MinX, c.Bounds().MinY+graphics.Scalar(line*c.lineHeight)-c.scrollY, c.Bounds().Width(), graphics.Scalar(c.lineHeight)),
		})
	}
	return nodes
}

func (c *EditorControl) accessibilityPerform(id string, action forms.AccessibilityAction, value string) bool {
	index, ok := accessibilityIndex(id, c.AccessibilityID()+"-completions-")
	if !ok || action != forms.AccessibilityActionInvoke || index < 0 || index >= len(c.completions) {
		return false
	}
	c.completionPick = index
	c.acceptCompletion()
	return true
}

func accessibilityIndex(id, prefix string) (int, bool) {
	if len(id) <= len(prefix) || !ideHasPrefix(id, prefix) {
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

func ideHasPrefix(text, prefix string) bool {
	if len(prefix) > len(text) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if text[i] != prefix[i] {
			return false
		}
	}
	return true
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
