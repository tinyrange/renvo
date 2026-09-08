package terminal

// ResizeToSurface refits the grid after a display orientation/size change.
// Recent lines and their colors are retained without reflow; columns beyond
// the new width are truncated. Storage is reused on subsequent switches.
// Flush invokes this automatically when the surface's logical dimensions change.
func (t *Terminal) ResizeToSurface() bool {
	if t.surface == nil || t.cellWidth <= 0 || t.cellHeight <= 0 {
		return false
	}
	contentHeight := t.surface.Height - t.keyboardHeight
	columns, rows := t.surface.Width/t.cellWidth, contentHeight/t.cellHeight
	if columns < 1 || rows < 1 {
		return false
	}
	capacity := t.capacity - t.rows + rows
	count := capacity * columns * 2
	storage := t.resizeCells
	if cap(storage) < count {
		storage = make([]uint32, count)
	} else {
		storage = storage[:count]
	}
	blank1, blank2 := t.blankCell()
	for i := 0; i < count; i += 2 {
		storage[i], storage[i+1] = blank1, blank2
	}
	keep := t.lineCount
	if keep > capacity {
		keep = capacity
	}
	drop := t.lineCount - keep
	width := columns
	if width > t.columns {
		width = t.columns
	}
	for line := 0; line < keep; line++ {
		from := t.cellIndex(line+drop, 0)
		to := line * columns * 2
		copy(storage[to:to+width*2], t.cells[from:from+width*2])
	}
	t.resizeCells, t.cells = t.cells, storage
	t.columns, t.rows, t.capacity, t.first = columns, rows, capacity, 0
	t.lineCount = keep
	if t.lineCount < rows {
		t.lineCount = rows
	}
	t.screenStart = t.lineCount - rows
	t.cursorLine -= drop
	if t.cursorLine < t.screenStart {
		t.cursorLine = t.screenStart
	}
	if t.cursorLine >= t.lineCount {
		t.cursorLine = t.lineCount - 1
	}
	if t.column >= columns {
		t.column = columns - 1
	}
	t.savedLine, t.savedColumn = t.cursorLine, t.column
	t.viewOffset, t.pendingScroll = 0, 0
	t.pendingWrap = false
	if cap(t.dirtyStart) < rows {
		t.dirtyStart, t.dirtyEnd = make([]int, rows), make([]int, rows)
	} else {
		t.dirtyStart, t.dirtyEnd = t.dirtyStart[:rows], t.dirtyEnd[:rows]
	}
	t.contentHeight = contentHeight
	t.surfaceWidth, t.surfaceHeight = t.surface.Width, t.surface.Height
	t.keyboardPressed, t.pointerDown = -1, false
	t.keyboardDirty, t.keyboardFullDirty = t.touchKeyboard, t.touchKeyboard
	t.markAllDirty()
	t.surface.Clear(t.defaultBackground)
	return true
}
