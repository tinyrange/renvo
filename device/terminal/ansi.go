package terminal

import "renvo.dev/std/graphics"

var ansiColors = [16]graphics.Color{
	graphics.RGBA(0, 0, 0, 255),
	graphics.RGBA(205, 49, 49, 255),
	graphics.RGBA(13, 188, 121, 255),
	graphics.RGBA(229, 229, 16, 255),
	graphics.RGBA(36, 114, 200, 255),
	graphics.RGBA(188, 63, 188, 255),
	graphics.RGBA(17, 168, 205, 255),
	graphics.RGBA(229, 229, 229, 255),
	graphics.RGBA(102, 102, 102, 255),
	graphics.RGBA(241, 76, 76, 255),
	graphics.RGBA(35, 209, 139, 255),
	graphics.RGBA(245, 245, 67, 255),
	graphics.RGBA(59, 142, 234, 255),
	graphics.RGBA(214, 112, 214, 255),
	graphics.RGBA(41, 184, 219, 255),
	graphics.RGBA(255, 255, 255, 255),
}

func ansi256Color(index int) graphics.Color {
	if index < 0 {
		index = 0
	}
	if index < 16 {
		color := ansiColors[index]
		return color
	}
	if index < 232 {
		index -= 16
		red := index / 36
		green := index / 6 % 6
		blue := index % 6
		component := func(value int) byte {
			if value == 0 {
				return 0
			}
			return byte(55 + value*40)
		}
		return graphics.RGBA(component(red), component(green), component(blue), 255)
	}
	if index > 255 {
		index = 255
	}
	gray := byte(8 + (index-232)*10)
	return graphics.RGBA(gray, gray, gray, 255)
}

func (t *Terminal) parameter(index, fallback int) int {
	if index < 0 || index >= len(t.parameters) || !t.parameterPresent[index] {
		return fallback
	}
	return t.parameters[index]
}

func (t *Terminal) clearCSI() {
	t.escapeState = 0
	t.parameters = [16]int{}
	t.parameterPresent = [16]bool{}
	t.parameterIndex = 0
	t.privateCSI = false
}

func (t *Terminal) eraseLine(mode int) {
	start, end := t.column, t.columns
	if mode == 1 {
		start, end = 0, t.column+1
	} else if mode == 2 {
		start, end = 0, t.columns
	}
	for column := start; column < end; column++ {
		t.setBlankCell(t.cursorLine, column)
	}
}

func (t *Terminal) clearVisibleRow(row int) {
	if row < 0 || row >= t.rows {
		return
	}
	logical := t.screenStart + row
	for column := 0; column < t.columns; column++ {
		t.setBlankCell(logical, column)
	}
}

func (t *Terminal) eraseDisplay(mode int) {
	row := t.cursorLine - t.screenStart
	if mode == 2 || mode == 3 {
		for at := 0; at < t.rows; at++ {
			t.clearVisibleRow(at)
		}
		if mode == 3 {
			t.first, t.lineCount, t.screenStart = 0, t.rows, 0
			t.cursorLine = row
			t.viewOffset = 0
		}
		return
	}
	if mode == 1 {
		for at := 0; at < row; at++ {
			t.clearVisibleRow(at)
		}
		start := t.column + 1
		for column := 0; column < start; column++ {
			t.setBlankCell(t.cursorLine, column)
		}
		return
	}
	t.eraseLine(0)
	for at := row + 1; at < t.rows; at++ {
		t.clearVisibleRow(at)
	}
}

func (t *Terminal) setSGRColor(foreground bool, color graphics.Color) {
	if foreground {
		t.foreground = color
	} else {
		t.background = color
	}
}

func (t *Terminal) finishExtendedColor(at int, foreground bool) int {
	mode := t.parameter(at+1, -1)
	if mode == 5 && at+2 <= t.parameterIndex {
		t.setSGRColor(foreground, ansi256Color(t.parameter(at+2, 0)))
		return at + 2
	}
	if mode == 2 && at+4 <= t.parameterIndex {
		red := t.parameter(at+2, 0)
		green := t.parameter(at+3, 0)
		blue := t.parameter(at+4, 0)
		if red < 0 {
			red = 0
		} else if red > 255 {
			red = 255
		}
		if green < 0 {
			green = 0
		} else if green > 255 {
			green = 255
		}
		if blue < 0 {
			blue = 0
		} else if blue > 255 {
			blue = 255
		}
		t.setSGRColor(foreground, graphics.RGBA(byte(red), byte(green), byte(blue), 255))
		return at + 4
	}
	return at
}

func (t *Terminal) sgr() {
	if !t.parameterPresent[0] && t.parameterIndex == 0 {
		t.parameters[0] = 0
		t.parameterPresent[0] = true
	}
	for at := 0; at <= t.parameterIndex; at++ {
		value := t.parameter(at, 0)
		switch {
		case value == 0:
			t.foreground, t.background = t.defaultForeground, t.defaultBackground
			t.bold, t.underline, t.inverse = false, false, false
		case value == 1:
			t.bold = true
		case value == 4:
			t.underline = true
		case value == 7:
			t.inverse = true
		case value == 22:
			t.bold = false
		case value == 24:
			t.underline = false
		case value == 27:
			t.inverse = false
		case value >= 30 && value <= 37:
			t.foreground = ansiColors[value-30]
		case value == 38:
			at = t.finishExtendedColor(at, true)
		case value == 39:
			t.foreground = t.defaultForeground
		case value >= 40 && value <= 47:
			t.background = ansiColors[value-40]
		case value == 48:
			at = t.finishExtendedColor(at, false)
		case value == 49:
			t.background = t.defaultBackground
		case value >= 90 && value <= 97:
			t.foreground = ansiColors[8+value-90]
		case value >= 100 && value <= 107:
			t.background = ansiColors[8+value-100]
		}
	}
}

func (t *Terminal) scrollScreen(lines int) {
	if lines < 1 {
		lines = 1
	}
	for count := 0; count < lines; count++ {
		t.appendLine()
	}
}

func (t *Terminal) finishCSI(command byte) {
	amount := t.parameter(0, 1)
	row, column := t.Cursor()
	switch command {
	case 'A':
		t.move(row-amount, column)
	case 'B':
		t.move(row+amount, column)
	case 'C':
		t.move(row, column+amount)
	case 'D':
		t.move(row, column-amount)
	case 'E':
		t.move(row+amount, 0)
	case 'F':
		t.move(row-amount, 0)
	case 'G':
		t.move(row, t.parameter(0, 1)-1)
	case 'H', 'f':
		t.move(t.parameter(0, 1)-1, t.parameter(1, 1)-1)
	case 'd':
		t.move(t.parameter(0, 1)-1, column)
	case 'J':
		t.eraseDisplay(t.parameter(0, 0))
	case 'K':
		t.eraseLine(t.parameter(0, 0))
	case 'S':
		t.scrollScreen(amount)
	case 'T':
		t.Scroll(amount)
	case 'm':
		t.sgr()
	case 's':
		t.savedLine, t.savedColumn = row, column
	case 'u':
		t.move(t.savedLine, t.savedColumn)
	case 'h':
		if t.privateCSI && t.parameter(0, 0) == 25 {
			t.cursorVisible = true
			t.markCursorDirty()
		}
	case 'l':
		if t.privateCSI && t.parameter(0, 0) == 25 {
			t.markCursorDirty()
			t.cursorVisible = false
		}
	}
	t.clearCSI()
}

func (t *Terminal) writeEscape(value byte) {
	if t.escapeState == 1 {
		switch value {
		case '[':
			t.escapeState = 2
			return
		case '7':
			t.savedLine, t.savedColumn = t.Cursor()
		case '8':
			t.move(t.savedLine, t.savedColumn)
		case 'c':
			t.Reset()
		case 'D':
			t.lineFeed()
		case 'E':
			t.carriageReturn()
			t.lineFeed()
		case 'M':
			row, column := t.Cursor()
			t.move(row-1, column)
		}
		t.escapeState = 0
		return
	}
	if value == '?' && t.parameterIndex == 0 && !t.parameterPresent[0] {
		t.privateCSI = true
		return
	}
	if value >= '0' && value <= '9' {
		t.parameters[t.parameterIndex] = t.parameters[t.parameterIndex]*10 + int(value-'0')
		t.parameterPresent[t.parameterIndex] = true
		return
	}
	if value == ';' {
		if t.parameterIndex < len(t.parameters)-1 {
			t.parameterIndex++
		}
		return
	}
	if value >= 0x40 && value <= 0x7e {
		t.finishCSI(value)
		return
	}
	if value < 0x20 {
		t.writeControl(value)
		return
	}
	t.clearCSI()
}
