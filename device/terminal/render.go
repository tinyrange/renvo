package terminal

import "renvo.dev/std/graphics"

func (t *Terminal) resolvedColors(cell Cell) (graphics.Color, graphics.Color) {
	foreground, background := cell.Foreground, cell.Background
	if cell.Inverse {
		foreground, background = background, foreground
	}
	return foreground, background
}

func (t *Terminal) renderRow(row, first, end int) {
	logical := t.displayStart() + row
	if logical < 0 || logical >= t.lineCount {
		return
	}
	y := row * t.cellHeight
	t.surface.FillRect(graphics.R(
		graphics.Scalar(first*t.cellWidth), graphics.Scalar(y),
		graphics.Scalar((end-first)*t.cellWidth), graphics.Scalar(t.cellHeight),
	), t.defaultBackground)
	for column := first; column < end; column++ {
		cell := t.cell(logical, column)
		foreground, background := t.resolvedColors(cell)
		x := column * t.cellWidth
		cursor := t.cursorVisible && t.viewOffset == 0 && logical == t.cursorLine && column == t.column
		if cursor {
			background = t.cursorColor
		}
		if background != t.defaultBackground || cursor {
			t.surface.FillRect(graphics.R(
				graphics.Scalar(x), graphics.Scalar(y),
				graphics.Scalar(t.cellWidth), graphics.Scalar(t.cellHeight),
			), background)
		}
		if cell.Character != ' ' && cell.Character != 0 {
			text := [1]byte{cell.Character}
			baseline := graphics.Point{X: graphics.Scalar(x), Y: graphics.Scalar(y + t.baseline)}
			t.surface.DrawTextBytes(t.font, baseline, text[:], foreground)
			if cell.Bold {
				baseline.X++
				t.surface.DrawTextBytes(t.font, baseline, text[:], foreground)
			}
		}
		if cell.Underline {
			t.surface.FillRect(graphics.R(
				graphics.Scalar(x), graphics.Scalar(y+t.cellHeight-2),
				graphics.Scalar(t.cellWidth), 1,
			), foreground)
		}
	}
}

func (t *Terminal) hasDirtyRows() bool {
	for row := 0; row < len(t.dirtyStart); row++ {
		if t.dirtyStart[row] < t.dirtyEnd[row] {
			return true
		}
	}
	return false
}

func (t *Terminal) scrollSurfacePixels(lines int) bool {
	if lines <= 0 || t.surface == nil {
		return true
	}
	pixelHeight := t.rows * t.cellHeight
	shift := lines * t.cellHeight
	if shift >= pixelHeight {
		t.surface.FillRect(graphics.R(
			0, 0, graphics.Scalar(t.surface.Width), graphics.Scalar(pixelHeight),
		), t.defaultBackground)
		return true
	}
	accelerated := false
	if t.scrollDisplay != nil {
		accelerated = t.scrollDisplay.ScrollTerminal(t.surface, 0, pixelHeight, shift)
	}
	if !accelerated {
		if t.scrollDisplay != nil && !t.scrollDisplay.PrepareTerminal(t.surface, false) {
			return false
		}
		t.surface.BeginDamage(graphics.R(
			0, 0, graphics.Scalar(t.surface.Width), graphics.Scalar(pixelHeight),
		))
		start := shift * t.surface.Stride
		length := (pixelHeight - shift) * t.surface.Stride
		copy(t.surface.Pixels[:length], t.surface.Pixels[start:start+length])
	}
	t.surface.FillRect(graphics.R(
		0, graphics.Scalar(pixelHeight-shift),
		graphics.Scalar(t.surface.Width), graphics.Scalar(shift),
	), t.defaultBackground)
	if !accelerated {
		t.surface.EndDamage()
	}
	return true
}

// Flush renders changed terminal rows and presents the display. It returns
// false when the board rejects presentation.
func (t *Terminal) Flush() bool {
	if t.surface == nil || t.display == nil {
		return true
	}
	if !t.hasDirtyRows() && t.pendingScroll == 0 && !t.keyboardDirty {
		return true
	}
	start := uint32(0)
	if t.clock != nil {
		start = t.clock.Milliseconds()
	}
	appliedScroll := t.pendingScroll
	if t.scrollDisplay != nil && !t.scrollDisplay.PrepareTerminal(t.surface, appliedScroll != 0) {
		return false
	}
	if appliedScroll != 0 {
		// A pixel memmove replaces re-rasterizing every glyph in the viewport.
		// Accelerated displays already know the moved region, so their ordinary
		// damage remains limited to rows which the CPU actually redraws.
		if !t.scrollSurfacePixels(appliedScroll) {
			return false
		}
		t.pendingScroll = 0
	}
	// One damage declaration per contiguous band prevents text and background
	// primitives from becoming hundreds of synchronous DMA2D copies on Tab5.
	for row := 0; row < len(t.dirtyStart); {
		if t.dirtyStart[row] >= t.dirtyEnd[row] {
			row++
			continue
		}
		first := row
		minColumn, maxColumn := t.dirtyStart[row], t.dirtyEnd[row]
		for row < len(t.dirtyStart) && t.dirtyStart[row] < t.dirtyEnd[row] {
			if t.dirtyStart[row] < minColumn {
				minColumn = t.dirtyStart[row]
			}
			if t.dirtyEnd[row] > maxColumn {
				maxColumn = t.dirtyEnd[row]
			}
			row++
		}
		t.surface.BeginDamage(graphics.R(
			graphics.Scalar(minColumn*t.cellWidth), graphics.Scalar(first*t.cellHeight),
			graphics.Scalar((maxColumn-minColumn)*t.cellWidth), graphics.Scalar((row-first)*t.cellHeight),
		))
		for at := first; at < row; at++ {
			t.renderRow(at, t.dirtyStart[at], t.dirtyEnd[at])
		}
		t.surface.EndDamage()
	}
	if t.touchKeyboard && t.keyboardDirty {
		if t.keyboardFullDirty {
			keyboardTop := t.rows * t.cellHeight
			t.surface.BeginDamage(graphics.R(
				0, graphics.Scalar(keyboardTop),
				graphics.Scalar(t.surface.Width), graphics.Scalar(t.surface.Height-keyboardTop),
			))
			if keyboardTop < t.contentHeight {
				t.surface.FillRect(graphics.R(
					0, graphics.Scalar(keyboardTop),
					graphics.Scalar(t.surface.Width), graphics.Scalar(t.contentHeight-keyboardTop),
				), t.defaultBackground)
			}
			t.renderKeyboard()
			t.surface.EndDamage()
		} else {
			for index := 0; index < touchKeyCount; index++ {
				if !t.keyboardDirtyKeys[index] {
					continue
				}
				rect := t.keyboardRect(index)
				t.surface.BeginDamage(rect)
				t.renderKeyboardKey(index)
				t.surface.EndDamage()
			}
		}
	}
	if !t.display.PresentTerminal(t.surface) {
		if appliedScroll != 0 {
			t.markAllDirty()
		}
		return false
	}
	t.surface.ResetDirty()
	for row := 0; row < len(t.dirtyStart); row++ {
		t.dirtyStart[row] = t.columns
		t.dirtyEnd[row] = 0
	}
	t.keyboardDirty = false
	t.keyboardFullDirty = false
	for index := 0; index < touchKeyCount; index++ {
		t.keyboardDirtyKeys[index] = false
	}
	t.stats.Frames++
	if t.clock != nil {
		elapsed := t.clock.Milliseconds() - start
		t.stats.LastFrameMillis = elapsed
		t.stats.RenderMillis += uint64(elapsed)
	}
	return true
}

const touchKeyCount = 42

func (t *Terminal) keyboardRect(index int) graphics.Rect {
	top := t.contentHeight
	rowHeight := t.keyboardHeight / 5
	margin := t.surface.Width / 100
	if margin < 4 {
		margin = 4
	}
	gap := margin
	inner := t.surface.Width - margin*2
	row, startUnits, widthUnits := 0, 0, 2
	switch {
	case index < 10:
		row, startUnits = 0, index*2
	case index < 20:
		row, startUnits = 1, (index-10)*2
	case index < 29:
		row, startUnits = 2, 1+(index-20)*2
	case index == 29:
		row, startUnits, widthUnits = 3, 0, 3
	case index < 37:
		row, startUnits = 3, 3+(index-30)*2
	case index == 37:
		row, startUnits, widthUnits = 3, 17, 3
	case index == 38:
		row, startUnits, widthUnits = 4, 0, 3
	case index == 39:
		row, startUnits, widthUnits = 4, 3, 3
	case index == 40:
		row, startUnits, widthUnits = 4, 6, 8
	default:
		row, startUnits, widthUnits = 4, 14, 6
	}
	x0 := margin + startUnits*inner/20
	x1 := margin + (startUnits+widthUnits)*inner/20
	y0 := top + row*rowHeight
	y1 := top + (row+1)*rowHeight
	return graphics.R(
		graphics.Scalar(x0+gap/2), graphics.Scalar(y0+gap/2),
		graphics.Scalar(x1-x0-gap), graphics.Scalar(y1-y0-gap),
	)
}

func (t *Terminal) keyboardLabel(index int) string {
	if index < 10 {
		return "1234567890"[index : index+1]
	}
	if index < 20 {
		return "qwertyuiop"[index-10 : index-9]
	}
	if index < 29 {
		return "asdfghjkl"[index-20 : index-19]
	}
	if index == 29 {
		return "Shift"
	}
	if index < 37 {
		return "zxcvbnm"[index-30 : index-29]
	}
	switch index {
	case 37:
		return "Back"
	case 38:
		return "Esc"
	case 39:
		return "Tab"
	case 40:
		return "Space"
	default:
		return "Enter"
	}
}

func (t *Terminal) renderKeyboard() {
	keyboardBackground := graphics.RGBA(18, 28, 38, 255)
	t.surface.FillRect(graphics.R(
		0, graphics.Scalar(t.contentHeight),
		graphics.Scalar(t.surface.Width), graphics.Scalar(t.keyboardHeight),
	), keyboardBackground)
	for index := 0; index < touchKeyCount; index++ {
		t.renderKeyboardKey(index)
	}
}

func (t *Terminal) renderKeyboardKey(index int) {
	keyBackground := graphics.RGBA(45, 59, 72, 255)
	pressedBackground := graphics.RGBA(55, 155, 105, 255)
	shiftBackground := graphics.RGBA(58, 90, 130, 255)
	rect := t.keyboardRect(index)
	color := keyBackground
	if index == t.keyboardPressed {
		color = pressedBackground
	} else if index == 29 && t.keyboardShift {
		color = shiftBackground
	}
	t.surface.FillRect(rect, color)
	label := t.keyboardLabel(index)
	metrics := graphics.MeasureText(t.font, label)
	x := rect.MinX + (rect.Width()-metrics.Width)/2
	y := rect.MinY + (rect.Height()-graphics.Scalar(t.cellHeight))/2 + graphics.Scalar(t.baseline)
	t.surface.DrawText(t.font, graphics.Point{X: x, Y: y}, label, t.defaultForeground)
}

func (t *Terminal) markKeyboardKey(index int) {
	if index < 0 || index >= touchKeyCount {
		return
	}
	t.keyboardDirty = true
	t.keyboardDirtyKeys[index] = true
}

func pointInRect(x, y int, rect graphics.Rect) bool {
	return graphics.Scalar(x) >= rect.MinX && graphics.Scalar(x) < rect.MaxX &&
		graphics.Scalar(y) >= rect.MinY && graphics.Scalar(y) < rect.MaxY
}

func (t *Terminal) keyboardIndexAt(x, y int) int {
	if y < t.contentHeight {
		return -1
	}
	for index := 0; index < touchKeyCount; index++ {
		if pointInRect(x, y, t.keyboardRect(index)) {
			return index
		}
	}
	return -1
}

func (t *Terminal) commitTouchKey(index int) {
	if index < 0 || index >= touchKeyCount {
		return
	}
	if index == 29 {
		t.keyboardShift = !t.keyboardShift
		t.markKeyboardKey(29)
		return
	}
	value := byte(0)
	if index < 10 {
		value = "1234567890"[index]
	} else if index < 20 {
		value = "qwertyuiop"[index-10]
	} else if index < 29 {
		value = "asdfghjkl"[index-20]
	} else if index < 37 {
		value = "zxcvbnm"[index-30]
	} else {
		switch index {
		case 37:
			value = 8
		case 38:
			value = 27
		case 39:
			value = 9
		case 40:
			value = ' '
		case 41:
			value = '\n'
		}
	}
	if t.keyboardShift && value >= 'a' && value <= 'z' {
		value = value - 'a' + 'A'
		t.keyboardShift = false
		t.markKeyboardKey(29)
	}
	t.queueInput(value)
}

// Poll consumes one optional pointer sample, handles keyboard input or a
// vertical scroll gesture, and flushes any resulting damage.
func (t *Terminal) Poll() bool {
	if t.pointer == nil {
		return t.Flush()
	}
	x, y, pressed, ok := t.pointer.ReadPointer()
	if !ok {
		return false
	}
	if pressed {
		if !t.pointerDown {
			t.pointerDown = true
			t.pointerStartY, t.pointerLastY = y, y
		}
		t.pointerLastY = y
		index := t.keyboardIndexAt(x, y)
		if index != t.keyboardPressed {
			t.markKeyboardKey(t.keyboardPressed)
			t.keyboardPressed = index
			t.markKeyboardKey(t.keyboardPressed)
		}
	} else if t.pointerDown {
		t.pointerDown = false
		index := t.keyboardPressed
		t.markKeyboardKey(t.keyboardPressed)
		t.keyboardPressed = -1
		if t.pointerStartY < t.contentHeight {
			delta := t.pointerStartY - t.pointerLastY
			if delta >= t.cellHeight || delta <= -t.cellHeight {
				t.Scroll(delta / t.cellHeight)
			}
		} else {
			t.commitTouchKey(index)
		}
	}
	return t.Flush()
}
