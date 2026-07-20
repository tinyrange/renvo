package forms

import "renvo.dev/std/graphics"

const widgetRowHeight = 24

func widgetText(surface *graphics.Surface, font *graphics.Font, x, y graphics.Scalar, text string, color graphics.Color) {
	if font != nil && text != "" {
		surface.DrawText(font, graphics.Point{X: x, Y: y + font.Metrics.Ascent}, text, color)
	}
}

func widgetValue(value int) string {
	if value < 0 {
		return "-" + accessibilityDecimal(-value)
	}
	return accessibilityDecimal(value)
}

func widgetParseValue(text string) (int, bool) {
	if text == "" {
		return 0, false
	}
	sign, at := 1, 0
	if text[0] == '-' {
		sign, at = -1, 1
	}
	if at == len(text) {
		return 0, false
	}
	value := 0
	for ; at < len(text); at++ {
		if text[at] < '0' || text[at] > '9' {
			return 0, false
		}
		value = value*10 + int(text[at]-'0')
	}
	return value * sign, true
}

func widgetItemID(control *Control, index int) string {
	return control.AccessibilityID() + "-item-" + accessibilityDecimal(index+1)
}

func widgetItemIndex(id, prefix string, count int) int {
	if len(id) <= len(prefix) || id[:len(prefix)] != prefix {
		return -1
	}
	value := 0
	for i := len(prefix); i < len(id); i++ {
		if id[i] < '0' || id[i] > '9' {
			return -1
		}
		value = value*10 + int(id[i]-'0')
	}
	value--
	if value < 0 || value >= count {
		return -1
	}
	return value
}

func widgetSelectedItem(items []string, index int, fallback string) string {
	if index >= 0 && index < len(items) {
		return items[index]
	}
	return fallback
}

func widgetTypeAhead(items []string, start int, prefix string) int {
	if len(items) == 0 || prefix == "" {
		return -1
	}
	for count := 0; count < len(items); count++ {
		index := (start + count + 1 + len(items)) % len(items)
		if menuTextStartsWith(items[index], prefix) {
			return index
		}
	}
	return -1
}

func widgetInvalidateSelection(control *Control, oldIndex, newIndex int, top graphics.Scalar, count int) {
	if control == nil {
		return
	}
	if control.Form() == nil {
		control.Invalidate()
		return
	}
	bounds := control.Bounds()
	for pass := 0; pass < 2; pass++ {
		index := oldIndex
		if pass == 1 {
			index = newIndex
		}
		if index >= 0 && index < count {
			control.Form().Invalidate(graphics.R(bounds.MinX+1, bounds.MinY+top+graphics.Scalar(index*widgetRowHeight), bounds.Width()-2, widgetRowHeight))
		}
	}
}

// ComboBox is a compact single-selection list with a transient drop-down.
type ComboBox struct {
	Control
	font             *graphics.Font
	items            []string
	selectedIndex    int
	closedBounds     graphics.Rect
	droppedDown      bool
	dropAbove        bool
	maxDropDownItems int
	scrollIndex      int
	Changed          EventHandler
}

func NewComboBox() *ComboBox {
	box := &ComboBox{selectedIndex: -1, maxDropDownItems: 8}
	box.Control = *NewControl()
	box.Control.applyTheme = box.applyTheme
	box.Control.layoutBounds = box.setLayoutBounds
	box.applyTheme(LightTheme())
	box.SetAccessibilityRole(AccessibilityRoleList)
	box.SetCursor(graphics.CursorPointingHand)
	box.AccessibilityValue = box.accessibilityValue
	box.AccessibilityChildren = box.accessibilityChildren
	box.AccessibilityPerform = box.accessibilityPerform
	box.Paint = box.paint
	box.PointerUp = box.pointerUp
	box.PointerWheel = box.pointerWheel
	box.KeyDown = box.keyDown
	box.TextInput = box.textInput
	box.Control.dismiss = box.dismiss
	return box
}

func (b *ComboBox) applyTheme(theme Theme) { applyFieldTheme(&b.Control, theme) }

func (b *ComboBox) Bounds() graphics.Rect { return b.closedBounds }
func (b *ComboBox) SetBounds(bounds graphics.Rect) {
	if b == nil {
		return
	}
	b.Control.setPreferredBounds(bounds)
	if b.Form() != nil && b.Dock() != DockNone {
		b.Form().performLayout()
		return
	}
	b.setLayoutBounds(bounds)
}
func (b *ComboBox) setLayoutBounds(bounds graphics.Rect) {
	if b == nil || rectEqual(b.closedBounds, bounds) {
		return
	}
	b.closedBounds = bounds
	b.refreshBounds()
}
func (b *ComboBox) Font() *graphics.Font        { return b.font }
func (b *ComboBox) SetFont(font *graphics.Font) { b.font = font; b.Invalidate() }
func (b *ComboBox) AddItem(text string) {
	b.items = append(b.items, text)
	b.refreshBounds()
	b.AccessibilityChildrenChanged()
	b.Invalidate()
}
func (b *ComboBox) SelectedIndex() int   { return b.selectedIndex }
func (b *ComboBox) SelectedItem() string { return widgetSelectedItem(b.items, b.selectedIndex, "") }
func (b *ComboBox) DroppedDown() bool    { return b.droppedDown }
func (b *ComboBox) SetMaxDropDownItems(count int) {
	if b == nil || count <= 0 || b.maxDropDownItems == count {
		return
	}
	b.maxDropDownItems = count
	b.ensureSelectionVisible()
	b.refreshBounds()
}
func (b *ComboBox) SetDroppedDown(open bool) {
	if b == nil || len(b.items) == 0 {
		open = false
	}
	if b == nil || b.droppedDown == open {
		return
	}
	b.droppedDown = open
	b.Control.popup = open
	if open {
		b.ensureSelectionVisible()
	}
	b.refreshBounds()
	b.AccessibilityChanged()
	b.AccessibilityChildrenStateChanged()
}
func (b *ComboBox) SetSelectedIndex(index int) {
	if b == nil || index < -1 || index >= len(b.items) || b.selectedIndex == index {
		return
	}
	b.selectedIndex = index
	b.AccessibilityChanged()
	b.AccessibilityChildrenStateChanged()
	b.Invalidate()
	if b.Changed != nil {
		b.Changed()
	}
}
func (b *ComboBox) accessibilityValue() string {
	return widgetSelectedItem(b.items, b.selectedIndex, b.Text())
}
func (b *ComboBox) accessibilityChildren() []AccessibilityNode {
	return selectionNodes(&b.Control, b.items, b.selectedIndex, AccessibilityRoleListItem, 0)
}
func (b *ComboBox) accessibilityPerform(id string, action AccessibilityAction, value string) bool {
	index := widgetItemIndex(id, b.AccessibilityID()+"-item-", len(b.items))
	if index < 0 || action != AccessibilityActionInvoke {
		return false
	}
	b.SetSelectedIndex(index)
	b.SetDroppedDown(false)
	return true
}
func (b *ComboBox) pointerUp(x, y graphics.Scalar) {
	header := b.headerRect()
	if pointInRect(x, y, header) {
		b.SetDroppedDown(!b.droppedDown)
		return
	}
	if !b.droppedDown {
		return
	}
	drop := b.dropRect()
	if pointInRect(x, y, drop) {
		index := b.scrollIndex + int(y-drop.MinY-1)/widgetRowHeight
		if index >= 0 && index < len(b.items) {
			b.SetSelectedIndex(index)
		}
	}
	b.SetDroppedDown(false)
}
func (b *ComboBox) pointerWheel(x, y graphics.Scalar) {
	if !b.droppedDown || len(b.items) <= b.visibleDropRows() {
		return
	}
	if y > 0 {
		b.scrollIndex++
	} else if y < 0 {
		b.scrollIndex--
	}
	b.clampScroll()
	b.Invalidate()
}
func (b *ComboBox) keyDown(event graphics.Event) {
	if event.Key == graphics.KeyEscape && b.droppedDown {
		b.SetDroppedDown(false)
		return
	}
	if event.Key == graphics.KeyDown && event.Modifiers&graphics.ModifierAlt != 0 {
		b.SetDroppedDown(true)
		return
	}
	if event.Key == graphics.KeyUp && event.Modifiers&graphics.ModifierAlt != 0 {
		b.SetDroppedDown(false)
		return
	}
	if event.Key == graphics.KeyEnter {
		if b.droppedDown {
			b.SetDroppedDown(false)
		}
		return
	}
	if event.Key == graphics.KeySpace {
		b.SetDroppedDown(!b.droppedDown)
		return
	}
	if event.Key == graphics.KeyHome && len(b.items) > 0 {
		b.SetSelectedIndex(0)
	} else if event.Key == graphics.KeyEnd && len(b.items) > 0 {
		b.SetSelectedIndex(len(b.items) - 1)
	} else if event.Key == graphics.KeyDown && len(b.items) > 0 {
		next := b.selectedIndex + 1
		if next >= len(b.items) {
			next = len(b.items) - 1
		}
		b.SetSelectedIndex(next)
	} else if event.Key == graphics.KeyUp && len(b.items) > 0 {
		next := b.selectedIndex - 1
		if next < 0 {
			next = 0
		}
		b.SetSelectedIndex(next)
	} else {
		return
	}
	b.ensureSelectionVisible()
	if b.droppedDown {
		b.Invalidate()
	}
}
func (b *ComboBox) textInput(text string) {
	index := widgetTypeAhead(b.items, b.selectedIndex, text)
	if index >= 0 {
		b.SetSelectedIndex(index)
		b.ensureSelectionVisible()
	}
}
func (b *ComboBox) paint(surface *graphics.Surface) {
	bounds := b.closedBounds
	theme := controlTheme(&b.Control)
	surface.FillRect(bounds, b.Background())
	border := theme.Border
	if b.Hovered() || b.droppedDown {
		border = controlAccent(&b.Control)
	}
	surface.StrokeRect(bounds, 1, border)
	arrow := graphics.R(bounds.MaxX-28, bounds.MinY, 28, bounds.Height())
	surface.FillRect(arrow, theme.SurfaceRaised)
	foreground := controlForeground(&b.Control)
	surface.DrawLine(graphics.Point{X: arrow.MinX + 9, Y: arrow.MinY + arrow.Height()/2 - 2}, graphics.Point{X: arrow.MinX + 14, Y: arrow.MinY + arrow.Height()/2 + 3}, 1, foreground)
	surface.DrawLine(graphics.Point{X: arrow.MinX + 14, Y: arrow.MinY + arrow.Height()/2 + 3}, graphics.Point{X: arrow.MinX + 19, Y: arrow.MinY + arrow.Height()/2 - 2}, 1, foreground)
	widgetText(surface, b.font, bounds.MinX+7, bounds.MinY+(bounds.Height()-labelLineHeight(b.font))/2, b.accessibilityValue(), foreground)
	if b.droppedDown {
		drop := b.dropRectGlobal()
		surface.FillRect(drop, b.Background())
		surface.StrokeRect(drop, 1, theme.Border)
		surface.PushClipRect(graphics.R(drop.MinX+1, drop.MinY+1, drop.Width()-2, drop.Height()-2))
		rows := b.visibleDropRows()
		for row := 0; row < rows; row++ {
			index := b.scrollIndex + row
			y := drop.MinY + 1 + graphics.Scalar(row*widgetRowHeight)
			if index == b.selectedIndex {
				surface.FillRect(graphics.R(drop.MinX+1, y, drop.Width()-2, widgetRowHeight), theme.Selection)
			}
			widgetText(surface, b.font, drop.MinX+7, y+4, b.items[index], foreground)
		}
		surface.PopClip()
	}
}

func (b *ComboBox) dismiss() { b.SetDroppedDown(false) }
func (b *ComboBox) visibleDropRows() int {
	rows := len(b.items)
	if rows > b.maxDropDownItems {
		rows = b.maxDropDownItems
	}
	return rows
}
func (b *ComboBox) dropHeight() graphics.Scalar {
	return graphics.Scalar(b.visibleDropRows()*widgetRowHeight + 2)
}
func (b *ComboBox) refreshBounds() {
	if b == nil {
		return
	}
	actual := b.closedBounds
	b.dropAbove = false
	if b.droppedDown {
		height := b.dropHeight()
		if b.Form() != nil {
			_, formHeight := b.Form().Size()
			if actual.MaxY+height > graphics.Scalar(formHeight) && actual.MinY >= height {
				actual.MinY -= height
				b.dropAbove = true
			} else {
				actual.MaxY += height
			}
		} else {
			actual.MaxY += height
		}
	}
	b.Control.setBoundsCore(actual)
}
func (b *ComboBox) headerRect() graphics.Rect {
	return graphics.R(b.closedBounds.MinX-b.Control.bounds.MinX, b.closedBounds.MinY-b.Control.bounds.MinY, b.closedBounds.Width(), b.closedBounds.Height())
}
func (b *ComboBox) dropRect() graphics.Rect {
	header := b.headerRect()
	height := b.dropHeight()
	if b.dropAbove {
		return graphics.R(header.MinX, header.MinY-height, header.Width(), height)
	}
	return graphics.R(header.MinX, header.MaxY, header.Width(), height)
}
func (b *ComboBox) dropRectGlobal() graphics.Rect {
	drop := b.dropRect()
	return graphics.R(b.Control.bounds.MinX+drop.MinX, b.Control.bounds.MinY+drop.MinY, drop.Width(), drop.Height())
}
func (b *ComboBox) clampScroll() {
	maximum := len(b.items) - b.visibleDropRows()
	if maximum < 0 {
		maximum = 0
	}
	if b.scrollIndex < 0 {
		b.scrollIndex = 0
	}
	if b.scrollIndex > maximum {
		b.scrollIndex = maximum
	}
}
func (b *ComboBox) ensureSelectionVisible() {
	rows := b.visibleDropRows()
	if b.selectedIndex >= 0 {
		if b.selectedIndex < b.scrollIndex {
			b.scrollIndex = b.selectedIndex
		}
		if b.selectedIndex >= b.scrollIndex+rows {
			b.scrollIndex = b.selectedIndex - rows + 1
		}
	}
	b.clampScroll()
}

// ListBox is a vertically arranged single-selection list.
type ListBox struct {
	Control
	font          *graphics.Font
	items         []string
	selectedIndex int
	hoveredIndex  int
	Changed       EventHandler
}

func NewListBox() *ListBox {
	b := &ListBox{selectedIndex: -1, hoveredIndex: -1}
	b.Control = *NewControl()
	b.Control.applyTheme = b.applyTheme
	b.applyTheme(LightTheme())
	b.SetAccessibilityRole(AccessibilityRoleList)
	b.AccessibilityChildren = b.accessibilityChildren
	b.AccessibilityPerform = b.accessibilityPerform
	b.Paint = b.paint
	b.PointerUp = b.pointerUp
	b.PointerMove = b.pointerMove
	b.PointerLeave = b.pointerLeave
	b.KeyDown = b.keyDown
	b.TextInput = b.textInput
	return b
}
func (b *ProgressBar) applyTheme(theme Theme)  { applyRaisedTheme(&b.Control, theme) }
func (b *ListBox) applyTheme(theme Theme)      { applyFieldTheme(&b.Control, theme) }
func (b *ListBox) Font() *graphics.Font        { return b.font }
func (b *ListBox) SetFont(font *graphics.Font) { b.font = font; b.Invalidate() }
func (b *ListBox) AddItem(text string) {
	b.items = append(b.items, text)
	b.AccessibilityChildrenChanged()
	b.Invalidate()
}
func (b *ListBox) SelectedIndex() int   { return b.selectedIndex }
func (b *ListBox) SelectedItem() string { return widgetSelectedItem(b.items, b.selectedIndex, "") }
func (b *ListBox) SetSelectedIndex(index int) {
	if b == nil || index < -1 || index >= len(b.items) || b.selectedIndex == index {
		return
	}
	old := b.selectedIndex
	b.selectedIndex = index
	b.AccessibilityChanged()
	b.AccessibilityChildrenStateChanged()
	widgetInvalidateSelection(&b.Control, old, index, 0, len(b.items))
	if b.Changed != nil {
		b.Changed()
	}
}
func (b *ListBox) displayItems() []string {
	if len(b.items) == 0 && b.Text() != "" {
		return []string{b.Text()}
	}
	return b.items
}
func (b *ListBox) accessibilityChildren() []AccessibilityNode {
	return selectionNodes(&b.Control, b.displayItems(), b.selectedIndex, AccessibilityRoleListItem, 0)
}
func (b *ListBox) accessibilityPerform(id string, action AccessibilityAction, value string) bool {
	index := widgetItemIndex(id, b.AccessibilityID()+"-item-", len(b.items))
	if index < 0 || action != AccessibilityActionInvoke {
		return false
	}
	b.SetSelectedIndex(index)
	return true
}
func (b *ListBox) pointerUp(x, y graphics.Scalar) {
	index := int(y-2) / widgetRowHeight
	if index >= 0 && index < len(b.items) {
		b.SetSelectedIndex(index)
	}
}
func (b *ListBox) pointerMove(x, y graphics.Scalar) {
	index := int(y-2) / widgetRowHeight
	if index < 0 || index >= len(b.items) {
		index = -1
	}
	if b.hoveredIndex != index {
		old := b.hoveredIndex
		b.hoveredIndex = index
		widgetInvalidateSelection(&b.Control, old, index, 0, len(b.items))
	}
}
func (b *ListBox) pointerLeave() {
	old := b.hoveredIndex
	b.hoveredIndex = -1
	widgetInvalidateSelection(&b.Control, old, -1, 0, len(b.items))
}
func (b *ListBox) keyDown(event graphics.Event) {
	next := b.selectedIndex
	if event.Key == graphics.KeyDown {
		next++
	} else if event.Key == graphics.KeyUp {
		next--
	} else if event.Key == graphics.KeyHome {
		next = 0
	} else if event.Key == graphics.KeyEnd {
		next = len(b.items) - 1
	} else if event.Key == graphics.KeyPageDown {
		next += b.visibleRows()
	} else if event.Key == graphics.KeyPageUp {
		next -= b.visibleRows()
	} else {
		return
	}
	if next < 0 {
		next = 0
	}
	if next >= len(b.items) {
		next = len(b.items) - 1
	}
	b.SetSelectedIndex(next)
}

func (b *ListBox) textInput(text string) {
	index := widgetTypeAhead(b.items, b.selectedIndex, text)
	if index >= 0 {
		b.SetSelectedIndex(index)
	}
}

func (b *ListBox) visibleRows() int {
	rows := int(b.Bounds().Height()-2) / widgetRowHeight
	if rows < 1 {
		rows = 1
	}
	return rows
}
func (b *ListBox) paint(surface *graphics.Surface) {
	paintRows(surface, &b.Control, b.font, b.displayItems(), b.selectedIndex, b.hoveredIndex, 0)
}

func selectionNodes(control *Control, items []string, selected int, role AccessibilityRole, top graphics.Scalar) []AccessibilityNode {
	nodes := make([]AccessibilityNode, 0, len(items))
	bounds := control.Bounds()
	for i := 0; i < len(items); i++ {
		nodes = append(nodes, AccessibilityNode{ID: widgetItemID(control, i), Role: role, Name: items[i], Bounds: graphics.R(bounds.MinX+1, bounds.MinY+top+graphics.Scalar(i*widgetRowHeight), bounds.Width()-2, widgetRowHeight), Actions: AccessibilitySupportsInvoke, Selectable: true, Selected: i == selected})
	}
	return nodes
}
func paintRows(surface *graphics.Surface, control *Control, font *graphics.Font, items []string, selected, hovered int, top graphics.Scalar) {
	bounds := control.Bounds()
	theme := controlTheme(control)
	surface.FillRect(bounds, control.Background())
	surface.StrokeRect(bounds, 1, theme.Border)
	surface.PushClipRect(graphics.R(bounds.MinX+1, bounds.MinY+1, bounds.Width()-2, bounds.Height()-2))
	for i := 0; i < len(items); i++ {
		y := bounds.MinY + top + graphics.Scalar(i*widgetRowHeight)
		if i == selected {
			surface.FillRect(graphics.R(bounds.MinX+1, y, bounds.Width()-2, widgetRowHeight), theme.Selection)
		} else if i == hovered {
			surface.FillRect(graphics.R(bounds.MinX+1, y, bounds.Width()-2, widgetRowHeight), theme.Hover)
		}
		widgetText(surface, font, bounds.MinX+7, y+4, items[i], controlForeground(control))
	}
	surface.PopClip()
}

// ListView displays rows under named columns.
type ListView struct {
	Control
	font          *graphics.Font
	columns       []string
	columnWidths  []int
	rows          [][]string
	selectedIndex int
	hoveredIndex  int
	Changed       EventHandler
}

func NewListView() *ListView {
	v := &ListView{selectedIndex: -1, hoveredIndex: -1}
	v.Control = *NewControl()
	v.Control.applyTheme = v.applyTheme
	v.applyTheme(LightTheme())
	v.SetAccessibilityRole(AccessibilityRoleList)
	v.AccessibilityChildren = v.accessibilityChildren
	v.AccessibilityPerform = v.accessibilityPerform
	v.Paint = v.paint
	v.PointerUp = v.pointerUp
	v.PointerMove = v.pointerMove
	v.PointerLeave = v.pointerLeave
	v.KeyDown = v.keyDown
	v.TextInput = v.textInput
	return v
}
func (v *ListView) applyTheme(theme Theme)      { applyFieldTheme(&v.Control, theme) }
func (v *ListView) Font() *graphics.Font        { return v.font }
func (v *ListView) SetFont(font *graphics.Font) { v.font = font; v.Invalidate() }
func (v *ListView) AddColumn(text string) {
	v.columns = append(v.columns, text)
	v.columnWidths = append(v.columnWidths, 0)
	v.Invalidate()
}
func (v *ListView) SetColumnWidth(index, width int) {
	if index < 0 || index >= len(v.columnWidths) || width < 0 || v.columnWidths[index] == width {
		return
	}
	v.columnWidths[index] = width
	v.Invalidate()
}
func (v *ListView) AddRow(values []string) {
	row := make([]string, len(values))
	copy(row, values)
	v.rows = append(v.rows, row)
	v.AccessibilityChildrenChanged()
	v.Invalidate()
}
func (v *ListView) SelectedIndex() int { return v.selectedIndex }
func (v *ListView) SetSelectedIndex(index int) {
	if index < -1 || index >= len(v.rows) || v.selectedIndex == index {
		return
	}
	old := v.selectedIndex
	v.selectedIndex = index
	v.AccessibilityChildrenStateChanged()
	widgetInvalidateSelection(&v.Control, old, index, widgetRowHeight, len(v.rows))
	if v.Changed != nil {
		v.Changed()
	}
}
func (v *ListView) rowText(index int) string {
	if index < 0 || index >= len(v.rows) {
		return ""
	}
	out := ""
	for i := 0; i < len(v.rows[index]); i++ {
		if i > 0 {
			out += "  "
		}
		out += v.rows[index][i]
	}
	return out
}
func (v *ListView) accessibilityChildren() []AccessibilityNode {
	items := make([]string, len(v.rows))
	for i := 0; i < len(v.rows); i++ {
		items[i] = v.rowText(i)
	}
	return selectionNodes(&v.Control, items, v.selectedIndex, AccessibilityRoleListItem, widgetRowHeight)
}
func (v *ListView) accessibilityPerform(id string, action AccessibilityAction, value string) bool {
	index := widgetItemIndex(id, v.AccessibilityID()+"-item-", len(v.rows))
	if index < 0 || action != AccessibilityActionInvoke {
		return false
	}
	v.SetSelectedIndex(index)
	return true
}
func (v *ListView) pointerUp(x, y graphics.Scalar) {
	index := int(y-widgetRowHeight) / widgetRowHeight
	if index >= 0 && index < len(v.rows) {
		v.SetSelectedIndex(index)
	}
}
func (v *ListView) pointerMove(x, y graphics.Scalar) {
	index := int(y-widgetRowHeight) / widgetRowHeight
	if y < widgetRowHeight || index < 0 || index >= len(v.rows) {
		index = -1
	}
	if v.hoveredIndex != index {
		old := v.hoveredIndex
		v.hoveredIndex = index
		widgetInvalidateSelection(&v.Control, old, index, widgetRowHeight, len(v.rows))
	}
}
func (v *ListView) pointerLeave() {
	old := v.hoveredIndex
	v.hoveredIndex = -1
	widgetInvalidateSelection(&v.Control, old, -1, widgetRowHeight, len(v.rows))
}
func (v *ListView) keyDown(event graphics.Event) {
	next := v.selectedIndex
	if event.Key == graphics.KeyDown {
		next++
	} else if event.Key == graphics.KeyUp {
		next--
	} else if event.Key == graphics.KeyHome {
		next = 0
	} else if event.Key == graphics.KeyEnd {
		next = len(v.rows) - 1
	} else if event.Key == graphics.KeyPageDown {
		next += v.visibleRows()
	} else if event.Key == graphics.KeyPageUp {
		next -= v.visibleRows()
	} else {
		return
	}
	if next < 0 {
		next = 0
	}
	if next >= len(v.rows) {
		next = len(v.rows) - 1
	}
	v.SetSelectedIndex(next)
}
func (v *ListView) textInput(text string) {
	items := make([]string, len(v.rows))
	for i := 0; i < len(v.rows); i++ {
		items[i] = v.rowText(i)
	}
	index := widgetTypeAhead(items, v.selectedIndex, text)
	if index >= 0 {
		v.SetSelectedIndex(index)
	}
}
func (v *ListView) visibleRows() int {
	rows := int(v.Bounds().Height()-widgetRowHeight-2) / widgetRowHeight
	if rows < 1 {
		rows = 1
	}
	return rows
}
func (v *ListView) paint(surface *graphics.Surface) {
	bounds := v.Bounds()
	theme := controlTheme(&v.Control)
	surface.FillRect(bounds, v.Background())
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, bounds.Width(), widgetRowHeight), theme.SurfaceRaised)
	surface.StrokeRect(bounds, 1, theme.Border)
	for i := 0; i < len(v.columns); i++ {
		x := bounds.MinX + v.columnX(i, bounds.Width())
		widgetText(surface, v.font, x+6, bounds.MinY+4, v.columns[i], controlForeground(&v.Control))
		if i > 0 {
			surface.FillRect(graphics.R(x, bounds.MinY, 1, bounds.Height()), theme.Border)
		}
	}
	for i := 0; i < len(v.rows); i++ {
		y := bounds.MinY + widgetRowHeight + graphics.Scalar(i*widgetRowHeight)
		if i == v.selectedIndex {
			surface.FillRect(graphics.R(bounds.MinX+1, y, bounds.Width()-2, widgetRowHeight), theme.Selection)
		} else if i == v.hoveredIndex {
			surface.FillRect(graphics.R(bounds.MinX+1, y, bounds.Width()-2, widgetRowHeight), theme.Hover)
		}
		for j := 0; j < len(v.rows[i]); j++ {
			x := bounds.MinX + v.columnX(j, bounds.Width())
			widgetText(surface, v.font, x+6, y+4, v.rows[i][j], controlForeground(&v.Control))
		}
	}
}

func (v *ListView) columnWidth(index int, total graphics.Scalar) graphics.Scalar {
	if index < 0 || index >= len(v.columns) || len(v.columns) == 0 {
		return 0
	}
	fixed := 0
	automatic := 0
	for i := 0; i < len(v.columns); i++ {
		if i < len(v.columnWidths) && v.columnWidths[i] > 0 {
			fixed += v.columnWidths[i]
		} else {
			automatic++
		}
	}
	if index < len(v.columnWidths) && v.columnWidths[index] > 0 {
		return graphics.Scalar(v.columnWidths[index])
	}
	remaining := int(total) - fixed
	if remaining < automatic {
		remaining = automatic
	}
	return graphics.Scalar(remaining / automatic)
}
func (v *ListView) columnX(index int, total graphics.Scalar) graphics.Scalar {
	x := graphics.Scalar(0)
	for i := 0; i < index && i < len(v.columns); i++ {
		x += v.columnWidth(i, total)
	}
	return x
}

type TreeNode struct {
	Text     string
	Level    int
	Expanded bool
}

// TreeView displays an expandable hierarchical outline. Nodes use explicit
// indentation so generated designer code remains compact and readable.
type TreeView struct {
	Control
	font          *graphics.Font
	nodes         []TreeNode
	selectedIndex int
	hoveredIndex  int
	Changed       EventHandler
}

func NewTreeView() *TreeView {
	v := &TreeView{selectedIndex: -1, hoveredIndex: -1}
	v.Control = *NewControl()
	v.Control.applyTheme = v.applyTheme
	v.applyTheme(LightTheme())
	v.SetAccessibilityRole(AccessibilityRoleTree)
	v.AccessibilityChildren = v.accessibilityChildren
	v.AccessibilityPerform = v.accessibilityPerform
	v.Paint = v.paint
	v.PointerUp = v.pointerUp
	v.PointerMove = v.pointerMove
	v.PointerLeave = v.pointerLeave
	v.KeyDown = v.keyDown
	v.TextInput = v.textInput
	return v
}
func (v *TreeView) applyTheme(theme Theme)      { applyFieldTheme(&v.Control, theme) }
func (v *TreeView) Font() *graphics.Font        { return v.font }
func (v *TreeView) SetFont(font *graphics.Font) { v.font = font; v.Invalidate() }
func (v *TreeView) AddNode(text string, level int) {
	if level < 0 {
		level = 0
	}
	v.nodes = append(v.nodes, TreeNode{Text: text, Level: level, Expanded: true})
	v.AccessibilityChildrenChanged()
	v.Invalidate()
}
func (v *TreeView) SelectedIndex() int { return v.selectedIndex }
func (v *TreeView) SetSelectedIndex(index int) {
	if index < -1 || index >= len(v.nodes) || v.selectedIndex == index {
		return
	}
	v.selectedIndex = index
	v.AccessibilityChildrenStateChanged()
	v.Invalidate()
	if v.Changed != nil {
		v.Changed()
	}
}
func (v *TreeView) Expanded(index int) bool {
	return index >= 0 && index < len(v.nodes) && v.hasChildren(index) && v.nodes[index].Expanded
}
func (v *TreeView) SetExpanded(index int, expanded bool) {
	if index < 0 || index >= len(v.nodes) || !v.hasChildren(index) || v.nodes[index].Expanded == expanded {
		return
	}
	v.nodes[index].Expanded = expanded
	if !expanded && v.selectedIndex > index && v.isDescendant(v.selectedIndex, index) {
		v.selectedIndex = index
		if v.Changed != nil {
			v.Changed()
		}
	}
	v.AccessibilityChildrenStateChanged()
	v.Invalidate()
}
func (v *TreeView) Toggle(index int) {
	if v.hasChildren(index) {
		v.SetExpanded(index, !v.nodes[index].Expanded)
	}
}
func (v *TreeView) ExpandAll() {
	changed := false
	for i := 0; i < len(v.nodes); i++ {
		if v.hasChildren(i) && !v.nodes[i].Expanded {
			v.nodes[i].Expanded = true
			changed = true
		}
	}
	if changed {
		v.AccessibilityChildrenStateChanged()
		v.Invalidate()
	}
}
func (v *TreeView) CollapseAll() {
	changed := false
	for i := 0; i < len(v.nodes); i++ {
		if v.hasChildren(i) && v.nodes[i].Expanded {
			v.nodes[i].Expanded = false
			changed = true
		}
	}
	if v.selectedIndex >= 0 && !v.nodeVisible(v.selectedIndex) {
		v.selectedIndex = v.visibleAncestor(v.selectedIndex)
		if v.Changed != nil {
			v.Changed()
		}
	}
	if changed {
		v.AccessibilityChildrenStateChanged()
		v.Invalidate()
	}
}
func (v *TreeView) accessibilityChildren() []AccessibilityNode {
	nodes := make([]AccessibilityNode, 0, len(v.nodes))
	bounds := v.Bounds()
	row := 0
	for i := 0; i < len(v.nodes); i++ {
		visible := v.nodeVisible(i)
		description := "Level " + widgetValue(v.nodes[i].Level)
		if v.hasChildren(i) {
			if v.nodes[i].Expanded {
				description += ", expanded"
			} else {
				description += ", collapsed"
			}
		}
		parent := v.parentIndex(i)
		parentID := ""
		if parent >= 0 {
			parentID = widgetItemID(&v.Control, parent)
		}
		nodeBounds := graphics.R(bounds.MinX+1, bounds.MinY+graphics.Scalar(row*widgetRowHeight), bounds.Width()-2, widgetRowHeight)
		nodes = append(nodes, AccessibilityNode{ID: widgetItemID(&v.Control, i), ParentID: parentID, Role: AccessibilityRoleTreeItem, Name: v.nodes[i].Text, Description: description, Bounds: nodeBounds, Actions: AccessibilitySupportsInvoke, Hidden: !visible, Selectable: true, Selected: i == v.selectedIndex})
		if visible {
			row++
		}
	}
	return nodes
}
func (v *TreeView) accessibilityPerform(id string, action AccessibilityAction, value string) bool {
	index := widgetItemIndex(id, v.AccessibilityID()+"-item-", len(v.nodes))
	if index < 0 || action != AccessibilityActionInvoke {
		return false
	}
	v.SetSelectedIndex(index)
	if v.hasChildren(index) {
		v.Toggle(index)
	}
	return true
}
func (v *TreeView) pointerUp(x, y graphics.Scalar) {
	index := v.nodeAtVisibleRow(int(y) / widgetRowHeight)
	if index < 0 {
		return
	}
	disclosureX := graphics.Scalar(7 + v.nodes[index].Level*16)
	if v.hasChildren(index) && x >= disclosureX && x < disclosureX+12 {
		v.Toggle(index)
		return
	}
	v.SetSelectedIndex(index)
}
func (v *TreeView) pointerMove(x, y graphics.Scalar) {
	index := v.nodeAtVisibleRow(int(y) / widgetRowHeight)
	if v.hoveredIndex != index {
		v.hoveredIndex = index
		v.Invalidate()
	}
}
func (v *TreeView) pointerLeave() {
	if v.hoveredIndex >= 0 {
		v.hoveredIndex = -1
		v.Invalidate()
	}
}
func (v *TreeView) keyDown(event graphics.Event) {
	if len(v.nodes) == 0 {
		return
	}
	if v.selectedIndex < 0 || !v.nodeVisible(v.selectedIndex) {
		v.SetSelectedIndex(v.nodeAtVisibleRow(0))
		return
	}
	if event.Key == graphics.KeyUp {
		v.selectVisibleOffset(-1)
	} else if event.Key == graphics.KeyDown {
		v.selectVisibleOffset(1)
	} else if event.Key == graphics.KeyLeft {
		if v.Expanded(v.selectedIndex) {
			v.SetExpanded(v.selectedIndex, false)
		} else {
			parent := v.parentIndex(v.selectedIndex)
			if parent >= 0 {
				v.SetSelectedIndex(parent)
			}
		}
	} else if event.Key == graphics.KeyRight {
		if v.hasChildren(v.selectedIndex) && !v.Expanded(v.selectedIndex) {
			v.SetExpanded(v.selectedIndex, true)
		} else if v.hasChildren(v.selectedIndex) {
			v.SetSelectedIndex(v.selectedIndex + 1)
		}
	} else if event.Key == graphics.KeyEnter || event.Key == graphics.KeySpace {
		v.Toggle(v.selectedIndex)
	} else if event.Key == graphics.KeyHome {
		v.SetSelectedIndex(v.nodeAtVisibleRow(0))
	} else if event.Key == graphics.KeyEnd {
		v.SetSelectedIndex(v.nodeAtVisibleRow(v.visibleNodeCount() - 1))
	}
}
func (v *TreeView) textInput(text string) {
	if text == "" || len(v.nodes) == 0 {
		return
	}
	start := v.selectedIndex
	for count := 0; count < len(v.nodes); count++ {
		index := (start + count + 1 + len(v.nodes)) % len(v.nodes)
		if v.nodeVisible(index) && menuTextStartsWith(v.nodes[index].Text, text) {
			v.SetSelectedIndex(index)
			return
		}
	}
}
func (v *TreeView) paint(surface *graphics.Surface) {
	bounds := v.Bounds()
	theme := controlTheme(&v.Control)
	surface.FillRect(bounds, v.Background())
	surface.StrokeRect(bounds, 1, theme.Border)
	row := 0
	for i := 0; i < len(v.nodes); i++ {
		if !v.nodeVisible(i) {
			continue
		}
		y := bounds.MinY + graphics.Scalar(row*widgetRowHeight)
		if i == v.selectedIndex {
			surface.FillRect(graphics.R(bounds.MinX+1, y, bounds.Width()-2, widgetRowHeight), theme.Selection)
		} else if i == v.hoveredIndex {
			surface.FillRect(graphics.R(bounds.MinX+1, y, bounds.Width()-2, widgetRowHeight), theme.Hover)
		}
		x := bounds.MinX + 7 + graphics.Scalar(v.nodes[i].Level*16)
		if v.hasChildren(i) {
			drawChevronIcon(surface, x, y+8, v.nodes[i].Expanded, theme.MutedText)
			drawIcon(surface, IconFolder, x+14, y+4, controlAccent(&v.Control))
		} else {
			drawIcon(surface, IconFile, x+14, y+4, theme.MutedText)
		}
		widgetText(surface, v.font, x+36, y+4, v.nodes[i].Text, controlForeground(&v.Control))
		row++
	}
}

func (v *TreeView) hasChildren(index int) bool {
	return index >= 0 && index+1 < len(v.nodes) && v.nodes[index+1].Level > v.nodes[index].Level
}
func (v *TreeView) parentIndex(index int) int {
	if index <= 0 || index >= len(v.nodes) {
		return -1
	}
	level := v.nodes[index].Level
	for i := index - 1; i >= 0; i-- {
		if v.nodes[i].Level < level {
			return i
		}
	}
	return -1
}
func (v *TreeView) isDescendant(index, parent int) bool {
	if index <= parent || index >= len(v.nodes) || parent < 0 || parent >= len(v.nodes) {
		return false
	}
	for index >= 0 {
		index = v.parentIndex(index)
		if index == parent {
			return true
		}
	}
	return false
}
func (v *TreeView) nodeVisible(index int) bool {
	if index < 0 || index >= len(v.nodes) {
		return false
	}
	parent := v.parentIndex(index)
	for parent >= 0 {
		if !v.nodes[parent].Expanded {
			return false
		}
		parent = v.parentIndex(parent)
	}
	return true
}
func (v *TreeView) visibleAncestor(index int) int {
	for index >= 0 && !v.nodeVisible(index) {
		index = v.parentIndex(index)
	}
	return index
}
func (v *TreeView) visibleNodeCount() int {
	count := 0
	for i := 0; i < len(v.nodes); i++ {
		if v.nodeVisible(i) {
			count++
		}
	}
	return count
}
func (v *TreeView) nodeAtVisibleRow(row int) int {
	if row < 0 {
		return -1
	}
	visible := 0
	for i := 0; i < len(v.nodes); i++ {
		if !v.nodeVisible(i) {
			continue
		}
		if visible == row {
			return i
		}
		visible++
	}
	return -1
}
func (v *TreeView) visibleRow(index int) int {
	row := 0
	for i := 0; i < len(v.nodes); i++ {
		if !v.nodeVisible(i) {
			continue
		}
		if i == index {
			return row
		}
		row++
	}
	return -1
}
func (v *TreeView) selectVisibleOffset(offset int) {
	row := v.visibleRow(v.selectedIndex)
	if row < 0 {
		row = 0
	} else {
		row += offset
	}
	if row < 0 {
		row = 0
	}
	count := v.visibleNodeCount()
	if row >= count {
		row = count - 1
	}
	v.SetSelectedIndex(v.nodeAtVisibleRow(row))
}

// TabControl provides a selectable tab strip; forms place page controls explicitly.
type TabControl struct {
	Control
	font          *graphics.Font
	tabs          []string
	icons         []Icon
	selectedIndex int
	hoveredIndex  int
	Changed       EventHandler
}

func NewTabControl() *TabControl {
	c := &TabControl{selectedIndex: -1, hoveredIndex: -1}
	c.Control = *NewControl()
	c.Control.applyTheme = c.applyTheme
	c.applyTheme(LightTheme())
	c.SetAccessibilityRole(AccessibilityRoleList)
	c.SetCursor(graphics.CursorPointingHand)
	c.AccessibilityChildren = c.accessibilityChildren
	c.AccessibilityPerform = c.accessibilityPerform
	c.Paint = c.paint
	c.PointerUp = c.pointerUp
	c.PointerMove = c.pointerMove
	c.PointerLeave = c.pointerLeave
	c.KeyDown = c.keyDown
	return c
}
func (c *TabControl) applyTheme(theme Theme)      { applySurfaceTheme(&c.Control, theme) }
func (c *TabControl) Font() *graphics.Font        { return c.font }
func (c *TabControl) SetFont(font *graphics.Font) { c.font = font; c.Invalidate() }
func (c *TabControl) AddTab(text string) {
	c.AddTabWithIcon(text, IconNone)
}
func (c *TabControl) AddTabWithIcon(text string, icon Icon) {
	c.tabs = append(c.tabs, text)
	c.icons = append(c.icons, icon)
	if c.selectedIndex < 0 {
		c.selectedIndex = 0
	}
	c.AccessibilityChildrenChanged()
	c.Invalidate()
}
func (c *TabControl) SelectedIndex() int { return c.selectedIndex }
func (c *TabControl) SetSelectedIndex(index int) {
	if index < 0 || index >= len(c.tabs) || c.selectedIndex == index {
		return
	}
	old := c.selectedIndex
	c.selectedIndex = index
	c.AccessibilityChildrenStateChanged()
	if c.Form() == nil || len(c.tabs) == 0 {
		c.Invalidate()
	} else {
		bounds := c.Bounds()
		width := bounds.Width() / graphics.Scalar(len(c.tabs))
		for pass := 0; pass < 2; pass++ {
			tab := old
			if pass == 1 {
				tab = index
			}
			if tab >= 0 && tab < len(c.tabs) {
				c.Form().Invalidate(graphics.R(bounds.MinX+graphics.Scalar(tab)*width, bounds.MinY, width, bounds.Height()))
			}
		}
	}
	if c.Changed != nil {
		c.Changed()
	}
}
func (c *TabControl) displayTabs() []string {
	if len(c.tabs) == 0 && c.Text() != "" {
		return []string{c.Text()}
	}
	return c.tabs
}
func (c *TabControl) accessibilityChildren() []AccessibilityNode {
	return selectionNodes(&c.Control, c.displayTabs(), c.selectedIndex, AccessibilityRoleListItem, 0)
}
func (c *TabControl) accessibilityPerform(id string, action AccessibilityAction, value string) bool {
	index := widgetItemIndex(id, c.AccessibilityID()+"-item-", len(c.tabs))
	if index < 0 || action != AccessibilityActionInvoke {
		return false
	}
	c.SetSelectedIndex(index)
	return true
}
func (c *TabControl) pointerUp(x, y graphics.Scalar) {
	if len(c.tabs) == 0 {
		return
	}
	width := c.Bounds().Width() / graphics.Scalar(len(c.tabs))
	if width > 0 {
		c.SetSelectedIndex(int(x / width))
	}
}
func (c *TabControl) pointerMove(x, y graphics.Scalar) {
	index := -1
	if len(c.tabs) > 0 {
		width := c.Bounds().Width() / graphics.Scalar(len(c.tabs))
		if width > 0 {
			index = int(x / width)
			if index < 0 || index >= len(c.tabs) {
				index = -1
			}
		}
	}
	if c.hoveredIndex != index {
		c.hoveredIndex = index
		c.Invalidate()
	}
}
func (c *TabControl) pointerLeave() {
	if c.hoveredIndex >= 0 {
		c.hoveredIndex = -1
		c.Invalidate()
	}
}
func (c *TabControl) keyDown(event graphics.Event) {
	if len(c.tabs) == 0 {
		return
	}
	index := c.selectedIndex
	if index < 0 {
		index = 0
	}
	if event.Key == graphics.KeyLeft {
		index = (index - 1 + len(c.tabs)) % len(c.tabs)
	} else if event.Key == graphics.KeyRight {
		index = (index + 1) % len(c.tabs)
	} else if event.Key == graphics.KeyHome {
		index = 0
	} else if event.Key == graphics.KeyEnd {
		index = len(c.tabs) - 1
	} else {
		return
	}
	c.SetSelectedIndex(index)
}
func (c *TabControl) paint(surface *graphics.Surface) {
	bounds := c.Bounds()
	theme := controlTheme(&c.Control)
	surface.FillRect(bounds, c.Background())
	tabs := c.displayTabs()
	if len(tabs) == 0 {
		return
	}
	width := bounds.Width() / graphics.Scalar(len(tabs))
	for i := 0; i < len(tabs); i++ {
		tab := graphics.R(bounds.MinX+graphics.Scalar(i)*width, bounds.MinY, width, bounds.Height())
		if i == c.selectedIndex {
			surface.FillRect(tab, theme.Selection)
		} else if i == c.hoveredIndex {
			surface.FillRect(tab, theme.Hover)
		} else {
			surface.FillRect(tab, theme.SurfaceRaised)
		}
		if i > 0 {
			surface.FillRect(graphics.R(tab.MinX, tab.MinY, 1, tab.Height()), theme.Border)
		}
		textX := tab.MinX + 10
		if i < len(c.icons) && c.icons[i] != IconNone {
			drawIcon(surface, c.icons[i], textX, tab.MinY+(tab.Height()-15)/2, controlAccent(&c.Control))
			textX += 22
		}
		widgetText(surface, c.font, textX, tab.MinY+(tab.Height()-labelLineHeight(c.font))/2, tabs[i], controlForeground(&c.Control))
		if i == c.selectedIndex {
			surface.FillRect(graphics.R(tab.MinX, tab.MaxY-3, tab.Width(), 3), controlAccent(&c.Control))
		}
	}
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, bounds.Width(), 1), theme.Border)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MaxY-1, bounds.Width(), 1), theme.Border)
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, 1, bounds.Height()), theme.Border)
	surface.FillRect(graphics.R(bounds.MaxX-1, bounds.MinY, 1, bounds.Height()), theme.Border)
}

type ProgressBar struct {
	Control
	minimum int
	maximum int
	value   int
}

func NewProgressBar() *ProgressBar {
	b := &ProgressBar{maximum: 100}
	b.Control = *NewControl()
	b.Control.applyTheme = b.applyTheme
	b.applyTheme(LightTheme())
	b.SetTabStop(false)
	b.SetAccessibilityRole(AccessibilityRoleStatus)
	b.AccessibilityValue = b.accessibilityValue
	b.Paint = b.paint
	return b
}
func (b *ProgressBar) Minimum() int { return b.minimum }
func (b *ProgressBar) Maximum() int { return b.maximum }
func (b *ProgressBar) Value() int   { return b.value }
func (b *ProgressBar) SetRange(minimum, maximum int) {
	if maximum <= minimum {
		return
	}
	b.minimum = minimum
	b.maximum = maximum
	b.SetValue(b.value)
	b.AccessibilityChanged()
	b.Invalidate()
}
func (b *ProgressBar) SetValue(value int) {
	value = clampWidgetValue(value, b.minimum, b.maximum)
	if b.value == value {
		return
	}
	b.value = value
	b.AccessibilityChanged()
	b.Invalidate()
}
func (b *ProgressBar) accessibilityValue() string { return widgetValue(b.value) }
func (b *ProgressBar) paint(surface *graphics.Surface) {
	bounds := b.Bounds()
	theme := controlTheme(&b.Control)
	surface.FillRect(bounds, b.Background())
	amount := graphics.Scalar(0)
	if b.maximum > b.minimum {
		amount = bounds.Width() * graphics.Scalar(b.value-b.minimum) / graphics.Scalar(b.maximum-b.minimum)
	}
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, amount, bounds.Height()), controlAccent(&b.Control))
	surface.StrokeRect(bounds, 1, theme.Border)
}
func clampWidgetValue(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

type NumericUpDown struct {
	Control
	font      *graphics.Font
	minimum   int
	maximum   int
	value     int
	increment int
	Changed   EventHandler
}

func NewNumericUpDown() *NumericUpDown {
	n := &NumericUpDown{maximum: 100, increment: 1}
	n.Control = *NewControl()
	n.Control.applyTheme = n.applyTheme
	n.applyTheme(LightTheme())
	n.SetAccessibilityRole(AccessibilityRoleTextBox)
	n.AccessibilityValue = n.accessibilityValue
	n.AccessibilitySetValue = n.accessibilitySetValue
	n.Paint = n.paint
	n.PointerUp = n.pointerUp
	n.KeyDown = n.keyDown
	return n
}
func (n *NumericUpDown) applyTheme(theme Theme)      { applyFieldTheme(&n.Control, theme) }
func (n *NumericUpDown) Font() *graphics.Font        { return n.font }
func (n *NumericUpDown) SetFont(font *graphics.Font) { n.font = font; n.Invalidate() }
func (n *NumericUpDown) Value() int                  { return n.value }
func (n *NumericUpDown) SetRange(minimum, maximum int) {
	if maximum <= minimum {
		return
	}
	n.minimum = minimum
	n.maximum = maximum
	n.SetValue(n.value)
}
func (n *NumericUpDown) SetIncrement(value int) {
	if value > 0 {
		n.increment = value
	}
}
func (n *NumericUpDown) SetValue(value int) {
	value = clampWidgetValue(value, n.minimum, n.maximum)
	if n.value == value {
		return
	}
	n.value = value
	n.AccessibilityChanged()
	n.Invalidate()
	if n.Changed != nil {
		n.Changed()
	}
}
func (n *NumericUpDown) accessibilityValue() string { return widgetValue(n.value) }
func (n *NumericUpDown) accessibilitySetValue(text string) {
	if value, ok := widgetParseValue(text); ok {
		n.SetValue(value)
	}
}
func (n *NumericUpDown) pointerUp(x, y graphics.Scalar) {
	if x >= n.Bounds().Width()-28 {
		if y < n.Bounds().Height()/2 {
			n.SetValue(n.value + n.increment)
		} else {
			n.SetValue(n.value - n.increment)
		}
	}
}
func (n *NumericUpDown) keyDown(event graphics.Event) {
	if event.Key == graphics.KeyUp {
		n.SetValue(n.value + n.increment)
	} else if event.Key == graphics.KeyDown {
		n.SetValue(n.value - n.increment)
	} else if event.Key == graphics.KeyPageUp {
		n.SetValue(n.value + n.increment*10)
	} else if event.Key == graphics.KeyPageDown {
		n.SetValue(n.value - n.increment*10)
	} else if event.Key == graphics.KeyHome {
		n.SetValue(n.minimum)
	} else if event.Key == graphics.KeyEnd {
		n.SetValue(n.maximum)
	}
}
func (n *NumericUpDown) paint(surface *graphics.Surface) {
	bounds := n.Bounds()
	theme := controlTheme(&n.Control)
	surface.FillRect(bounds, n.Background())
	border := theme.Border
	if n.Hovered() {
		border = controlAccent(&n.Control)
	}
	surface.StrokeRect(bounds, 1, border)
	buttons := graphics.R(bounds.MaxX-28, bounds.MinY, 28, bounds.Height())
	surface.FillRect(buttons, theme.SurfaceRaised)
	surface.FillRect(graphics.R(buttons.MinX, buttons.MinY+buttons.Height()/2, buttons.Width(), 1), theme.Border)
	foreground := controlForeground(&n.Control)
	widgetText(surface, n.font, bounds.MinX+7, bounds.MinY+(bounds.Height()-labelLineHeight(n.font))/2, widgetValue(n.value), foreground)
	surface.DrawLine(graphics.Point{X: buttons.MinX + 9, Y: buttons.MinY + 11}, graphics.Point{X: buttons.MinX + 14, Y: buttons.MinY + 6}, 1, foreground)
	surface.DrawLine(graphics.Point{X: buttons.MinX + 14, Y: buttons.MinY + 6}, graphics.Point{X: buttons.MinX + 19, Y: buttons.MinY + 11}, 1, foreground)
}

type Slider struct {
	Control
	minimum     int
	maximum     int
	value       int
	smallChange int
	largeChange int
	dragging    bool
	Changed     EventHandler
}

func NewSlider() *Slider {
	s := &Slider{maximum: 100, smallChange: 1, largeChange: 10}
	s.Control = *NewControl()
	s.Control.applyTheme = s.applyTheme
	s.applyTheme(LightTheme())
	s.SetAccessibilityRole(AccessibilityRoleGroup)
	s.AccessibilityValue = s.accessibilityValue
	s.SetCursor(graphics.CursorPointingHand)
	s.Paint = s.paint
	s.PointerDown = s.pointerDown
	s.PointerMove = s.pointerMove
	s.PointerUp = s.pointerUp
	s.KeyDown = s.keyDown
	return s
}
func (s *Slider) applyTheme(theme Theme) { applyTransparentTheme(&s.Control, theme) }
func (s *Slider) Value() int             { return s.value }
func (s *Slider) SetSmallChange(value int) {
	if value > 0 {
		s.smallChange = value
	}
}
func (s *Slider) SetLargeChange(value int) {
	if value > 0 {
		s.largeChange = value
	}
}
func (s *Slider) SetRange(minimum, maximum int) {
	if maximum <= minimum {
		return
	}
	s.minimum = minimum
	s.maximum = maximum
	s.SetValue(s.value)
}
func (s *Slider) SetValue(value int) {
	value = clampWidgetValue(value, s.minimum, s.maximum)
	if s.value == value {
		return
	}
	s.value = value
	s.AccessibilityChanged()
	s.Invalidate()
	if s.Changed != nil {
		s.Changed()
	}
}
func (s *Slider) accessibilityValue() string { return widgetValue(s.value) }
func (s *Slider) updateFromPointer(x graphics.Scalar) {
	width := s.Bounds().Width() - 16
	if width <= 0 {
		return
	}
	value := s.minimum + int((x-8)*graphics.Scalar(s.maximum-s.minimum)/width)
	s.SetValue(value)
}
func (s *Slider) pointerDown(x, y graphics.Scalar) {
	s.dragging = true
	s.updateFromPointer(x)
}
func (s *Slider) pointerMove(x, y graphics.Scalar) {
	if s.dragging {
		s.updateFromPointer(x)
	}
}
func (s *Slider) pointerUp(x, y graphics.Scalar) {
	if s.dragging {
		s.updateFromPointer(x)
		s.dragging = false
		s.Invalidate()
	}
}
func (s *Slider) keyDown(event graphics.Event) {
	if event.Key == graphics.KeyLeft || event.Key == graphics.KeyDown {
		s.SetValue(s.value - s.smallChange)
	} else if event.Key == graphics.KeyRight || event.Key == graphics.KeyUp {
		s.SetValue(s.value + s.smallChange)
	} else if event.Key == graphics.KeyPageDown {
		s.SetValue(s.value - s.largeChange)
	} else if event.Key == graphics.KeyPageUp {
		s.SetValue(s.value + s.largeChange)
	} else if event.Key == graphics.KeyHome {
		s.SetValue(s.minimum)
	} else if event.Key == graphics.KeyEnd {
		s.SetValue(s.maximum)
	}
}
func (s *Slider) paint(surface *graphics.Surface) {
	bounds := s.Bounds()
	theme := controlTheme(&s.Control)
	y := bounds.MinY + bounds.Height()/2
	surface.FillRect(graphics.R(bounds.MinX+8, y-2, bounds.Width()-16, 4), theme.Border)
	x := bounds.MinX + 8
	if s.maximum > s.minimum {
		x += (bounds.Width() - 16) * graphics.Scalar(s.value-s.minimum) / graphics.Scalar(s.maximum-s.minimum)
	}
	knob := graphics.R(x-7, y-7, 14, 14)
	if s.Hovered() || s.dragging {
		surface.FillEllipse(graphics.R(x-10, y-10, 20, 20), theme.Hover)
	}
	surface.FillEllipse(knob, controlAccent(&s.Control))
}

type GroupBox struct {
	Control
	font *graphics.Font
}

func NewGroupBox() *GroupBox {
	g := &GroupBox{}
	g.Control = *NewControl()
	g.Control.applyTheme = g.applyTheme
	g.applyTheme(LightTheme())
	g.SetTabStop(false)
	g.SetAccessibilityRole(AccessibilityRoleGroup)
	g.Paint = g.paint
	return g
}
func (g *GroupBox) applyTheme(theme Theme)      { applyTransparentTheme(&g.Control, theme) }
func (g *GroupBox) Font() *graphics.Font        { return g.font }
func (g *GroupBox) SetFont(font *graphics.Font) { g.font = font; g.Invalidate() }
func (g *GroupBox) paint(surface *graphics.Surface) {
	bounds := g.Bounds()
	theme := controlTheme(&g.Control)
	top := bounds.MinY + 8
	surface.StrokeRect(graphics.R(bounds.MinX, top, bounds.Width(), bounds.Height()-8), 1, theme.Border)
	if g.Text() != "" {
		width := graphics.Scalar(len(g.Text())*8 + 12)
		if g.font != nil {
			width = graphics.MeasureText(g.font, g.Text()).Width + 12
		}
		surface.FillRect(graphics.R(bounds.MinX+8, bounds.MinY, width, 18), theme.Window)
		widgetText(surface, g.font, bounds.MinX+14, bounds.MinY, g.Text(), controlForeground(&g.Control))
	}
}

type SplitContainer struct {
	Control
	splitterDistance int
	panel1MinSize    int
	panel2MinSize    int
	vertical         bool
	dragging         bool
	hoverSplitter    bool
	Changed          EventHandler
}

func NewSplitContainer() *SplitContainer {
	s := &SplitContainer{splitterDistance: 100, panel1MinSize: 24, panel2MinSize: 24, vertical: true}
	s.Control = *NewControl()
	s.Control.applyTheme = s.applyTheme
	s.applyTheme(LightTheme())
	s.SetTabStop(false)
	s.SetAccessibilityRole(AccessibilityRoleGroup)
	s.Paint = s.paint
	s.PointerDown = s.pointerDown
	s.PointerMove = s.pointerMove
	s.PointerUp = s.pointerUp
	s.PointerLeave = s.pointerLeave
	s.PointerCursor = s.pointerCursor
	return s
}
func (s *SplitContainer) applyTheme(theme Theme) { applySurfaceTheme(&s.Control, theme) }
func (s *SplitContainer) SplitterDistance() int  { return s.splitterDistance }
func (s *SplitContainer) SetPanelMinimumSizes(panel1, panel2 int) {
	if panel1 < 0 || panel2 < 0 {
		return
	}
	s.panel1MinSize = panel1
	s.panel2MinSize = panel2
	s.SetSplitterDistance(s.splitterDistance)
}
func (s *SplitContainer) SetSplitterDistance(value int) {
	limit := int(s.Bounds().Width())
	if !s.vertical {
		limit = int(s.Bounds().Height())
	}
	if value < s.panel1MinSize {
		value = s.panel1MinSize
	}
	maximum := limit - s.panel2MinSize
	if maximum < s.panel1MinSize {
		maximum = s.panel1MinSize
	}
	if value > maximum {
		value = maximum
	}
	if s.splitterDistance == value {
		return
	}
	s.splitterDistance = value
	s.Invalidate()
	if s.Changed != nil {
		s.Changed()
	}
}
func (s *SplitContainer) SetVertical(vertical bool) {
	if s.vertical != vertical {
		s.vertical = vertical
		s.Invalidate()
	}
}
func (s *SplitContainer) splitterPosition(x, y graphics.Scalar) int {
	if s.vertical {
		return int(x)
	}
	return int(y)
}
func (s *SplitContainer) pointerCursor(x, y graphics.Scalar) graphics.Cursor {
	if !s.overSplitter(x, y) {
		return graphics.CursorArrow
	}
	if s.vertical {
		return graphics.CursorResizeHorizontal
	}
	return graphics.CursorResizeVertical
}
func (s *SplitContainer) overSplitter(x, y graphics.Scalar) bool {
	position := s.splitterPosition(x, y)
	return position >= s.splitterDistance-5 && position <= s.splitterDistance+5
}
func (s *SplitContainer) pointerDown(x, y graphics.Scalar) {
	if s.overSplitter(x, y) {
		s.dragging = true
		s.SetSplitterDistance(s.splitterPosition(x, y))
	}
}
func (s *SplitContainer) pointerMove(x, y graphics.Scalar) {
	hover := s.overSplitter(x, y)
	if s.hoverSplitter != hover {
		s.hoverSplitter = hover
		s.Invalidate()
	}
	if s.dragging {
		s.SetSplitterDistance(s.splitterPosition(x, y))
	}
}
func (s *SplitContainer) pointerUp(x, y graphics.Scalar) {
	if s.dragging {
		s.SetSplitterDistance(s.splitterPosition(x, y))
		s.dragging = false
		s.Invalidate()
	}
}
func (s *SplitContainer) pointerLeave() {
	if s.hoverSplitter && !s.dragging {
		s.hoverSplitter = false
		s.Invalidate()
	}
}
func (s *SplitContainer) paint(surface *graphics.Surface) {
	bounds := s.Bounds()
	theme := controlTheme(&s.Control)
	surface.FillRect(bounds, s.Background())
	surface.StrokeRect(bounds, 1, theme.Border)
	color := theme.Border
	if s.hoverSplitter || s.dragging {
		color = controlAccent(&s.Control)
	}
	if s.vertical {
		surface.FillRect(graphics.R(bounds.MinX+graphics.Scalar(s.splitterDistance)-2, bounds.MinY, 5, bounds.Height()), color)
	} else {
		surface.FillRect(graphics.R(bounds.MinX, bounds.MinY+graphics.Scalar(s.splitterDistance)-2, bounds.Width(), 5), color)
	}
}

type toolBarItem struct {
	Text     string
	Icon     Icon
	activate EventHandler
}
type ToolBar struct {
	Control
	font         *graphics.Font
	items        []toolBarItem
	hoveredIndex int
}

func NewToolBar() *ToolBar {
	b := &ToolBar{hoveredIndex: -1}
	b.Control = *NewControl()
	b.Control.applyTheme = b.applyTheme
	b.applyTheme(LightTheme())
	b.SetTabStop(false)
	b.SetAccessibilityRole(AccessibilityRoleGroup)
	b.AccessibilityChildren = b.accessibilityChildren
	b.AccessibilityPerform = b.accessibilityPerform
	b.Paint = b.paint
	b.PointerUp = b.pointerUp
	b.PointerMove = b.pointerMove
	b.PointerLeave = b.pointerLeave
	return b
}
func (b *ToolBar) applyTheme(theme Theme)      { applyRaisedTheme(&b.Control, theme) }
func (b *ToolBar) Font() *graphics.Font        { return b.font }
func (b *ToolBar) SetFont(font *graphics.Font) { b.font = font; b.Invalidate() }
func (b *ToolBar) AddButton(text string, activate EventHandler) {
	b.AddButtonWithIcon(text, IconNone, activate)
}
func (b *ToolBar) AddButtonWithIcon(text string, icon Icon, activate EventHandler) {
	b.items = append(b.items, toolBarItem{Text: text, Icon: icon, activate: activate})
	b.AccessibilityChildrenChanged()
	b.Invalidate()
}
func (b *ToolBar) itemWidth(index int) graphics.Scalar {
	if index < 0 || index >= len(b.items) {
		return 0
	}
	width := graphics.Scalar(len(b.items[index].Text)*8 + 24)
	if b.font != nil {
		width = graphics.MeasureText(b.font, b.items[index].Text).Width + 24
	}
	if width < 64 {
		width = 64
	}
	if b.items[index].Icon != IconNone {
		width += 22
	}
	return width
}
func (b *ToolBar) itemAt(x graphics.Scalar) int {
	at := graphics.Scalar(4)
	for i := 0; i < len(b.items); i++ {
		width := b.itemWidth(i)
		if x >= at && x < at+width {
			return i
		}
		at += width + 4
	}
	return -1
}
func (b *ToolBar) pointerUp(x, y graphics.Scalar) {
	index := b.itemAt(x)
	if index >= 0 {
		item := b.items[index]
		if item.activate != nil {
			item.activate()
		}
	}
}
func (b *ToolBar) pointerMove(x, y graphics.Scalar) {
	index := b.itemAt(x)
	if b.hoveredIndex != index {
		b.hoveredIndex = index
		b.Invalidate()
	}
}
func (b *ToolBar) pointerLeave() {
	if b.hoveredIndex >= 0 {
		b.hoveredIndex = -1
		b.Invalidate()
	}
}
func (b *ToolBar) accessibilityChildren() []AccessibilityNode {
	nodes := make([]AccessibilityNode, 0, len(b.items))
	bounds := b.Bounds()
	x := bounds.MinX + 4
	for i := 0; i < len(b.items); i++ {
		width := b.itemWidth(i)
		nodes = append(nodes, AccessibilityNode{ID: widgetItemID(&b.Control, i), Role: AccessibilityRoleButton, Name: b.items[i].Text, Bounds: graphics.R(x, bounds.MinY+3, width, bounds.Height()-6), Actions: AccessibilitySupportsInvoke})
		x += width + 4
	}
	return nodes
}
func (b *ToolBar) accessibilityPerform(id string, action AccessibilityAction, value string) bool {
	index := widgetItemIndex(id, b.AccessibilityID()+"-item-", len(b.items))
	if index < 0 || action != AccessibilityActionInvoke {
		return false
	}
	item := b.items[index]
	if item.activate != nil {
		item.activate()
	}
	return true
}
func (b *ToolBar) paint(surface *graphics.Surface) {
	bounds := b.Bounds()
	theme := controlTheme(&b.Control)
	surface.FillRect(bounds, b.Background())
	surface.StrokeRect(bounds, 1, theme.Border)
	x := bounds.MinX + 4
	if len(b.items) == 0 && b.Text() != "" {
		widgetText(surface, b.font, x+8, bounds.MinY+(bounds.Height()-labelLineHeight(b.font))/2, b.Text(), controlForeground(&b.Control))
	}
	for i := 0; i < len(b.items); i++ {
		width := b.itemWidth(i)
		item := graphics.R(x, bounds.MinY+3, width, bounds.Height()-6)
		fill := theme.Surface
		if i == b.hoveredIndex {
			fill = theme.Hover
		}
		surface.FillRect(item, fill)
		surface.StrokeRect(item, 1, theme.Border)
		textX := item.MinX + 12
		if b.items[i].Icon != IconNone {
			drawIcon(surface, b.items[i].Icon, item.MinX+10, item.MinY+(item.Height()-15)/2, controlAccent(&b.Control))
			textX += 22
		}
		widgetText(surface, b.font, textX, item.MinY+(item.Height()-labelLineHeight(b.font))/2, b.items[i].Text, controlForeground(&b.Control))
		x += width + 4
	}
}

type StatusBar struct {
	Control
	font *graphics.Font
}

func NewStatusBar() *StatusBar {
	s := &StatusBar{}
	s.Control = *NewControl()
	s.Control.applyTheme = s.applyTheme
	s.applyTheme(LightTheme())
	s.SetTabStop(false)
	s.SetAccessibilityRole(AccessibilityRoleStatus)
	s.Paint = s.paint
	return s
}
func (s *StatusBar) applyTheme(theme Theme)      { applyRaisedTheme(&s.Control, theme) }
func (s *StatusBar) Font() *graphics.Font        { return s.font }
func (s *StatusBar) SetFont(font *graphics.Font) { s.font = font; s.Invalidate() }
func (s *StatusBar) paint(surface *graphics.Surface) {
	bounds := s.Bounds()
	surface.FillRect(bounds, s.Background())
	surface.FillRect(graphics.R(bounds.MinX, bounds.MinY, bounds.Width(), 1), controlTheme(&s.Control).Border)
	widgetText(surface, s.font, bounds.MinX+8, bounds.MinY+(bounds.Height()-labelLineHeight(s.font))/2, s.Text(), controlForeground(&s.Control))
}
