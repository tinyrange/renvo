package main

const (
	terminalColumns = 20
	terminalRows    = 6
)

// terminal is a deliberately small ASCII terminal. It implements the C0
// controls useful for local keyboard input and the common ANSI cursor/erase
// sequences without allocating while characters are being processed.
type terminal struct {
	cells            [terminalRows][terminalColumns]byte
	row, column      int
	escapeState      byte
	parameters       [2]int
	parameterPresent [2]bool
	parameterIndex   int
	bell             bool
}

func newTerminal() terminal {
	var t terminal
	t.clear()
	return t
}

func (t *terminal) clearRow(row int) {
	for column := 0; column < terminalColumns; column++ {
		t.cells[row][column] = ' '
	}
}

func (t *terminal) clear() {
	for row := 0; row < terminalRows; row++ {
		t.clearRow(row)
	}
	t.row = 0
	t.column = 0
}

func (t *terminal) scroll() {
	for row := 1; row < terminalRows; row++ {
		t.cells[row-1] = t.cells[row]
	}
	t.clearRow(terminalRows - 1)
}

func (t *terminal) lineFeed() {
	if t.row == terminalRows-1 {
		t.scroll()
	} else {
		t.row++
	}
}

func (t *terminal) put(value byte) {
	if value < 0x20 || value >= 0x7f {
		return
	}
	t.cells[t.row][t.column] = value
	t.column++
	if t.column == terminalColumns {
		t.column = 0
		t.lineFeed()
	}
}

func (t *terminal) backspace() {
	if t.column > 0 {
		t.column--
	} else if t.row > 0 {
		t.row--
		t.column = terminalColumns - 1
	}
	t.cells[t.row][t.column] = ' '
}

func (t *terminal) tab() {
	stop := (t.column + 4) &^ 3
	if stop >= terminalColumns {
		t.column = 0
		t.lineFeed()
		return
	}
	for t.column < stop {
		t.put(' ')
	}
}

func (t *terminal) move(deltaRow, deltaColumn int) {
	t.row += deltaRow
	t.column += deltaColumn
	if t.row < 0 {
		t.row = 0
	} else if t.row >= terminalRows {
		t.row = terminalRows - 1
	}
	if t.column < 0 {
		t.column = 0
	} else if t.column >= terminalColumns {
		t.column = terminalColumns - 1
	}
}

func (t *terminal) eraseLine(mode int) {
	switch mode {
	case 1:
		for column := 0; column <= t.column; column++ {
			t.cells[t.row][column] = ' '
		}
	case 2:
		t.clearRow(t.row)
	default:
		for column := t.column; column < terminalColumns; column++ {
			t.cells[t.row][column] = ' '
		}
	}
}

func (t *terminal) finishCSI(command byte) {
	value := t.parameters[t.parameterIndex]
	if !t.parameterPresent[t.parameterIndex] {
		value = 1
	}
	switch command {
	case 'A':
		t.move(-value, 0)
	case 'B':
		t.move(value, 0)
	case 'C':
		t.move(0, value)
	case 'D':
		t.move(0, -value)
	case 'H', 'f':
		row, column := 1, 1
		if t.parameterPresent[0] {
			row = t.parameters[0]
		}
		if t.parameterPresent[1] {
			column = t.parameters[1]
		}
		t.row = row - 1
		t.column = column - 1
		t.move(0, 0)
	case 'J':
		if value == 2 || !t.parameterPresent[0] {
			t.clear()
		}
	case 'K':
		t.eraseLine(value)
	case 'm':
		// SGR is accepted and ignored; this monochrome terminal preserves the
		// stream position while remaining allocation-free.
	}
	t.escapeState = 0
	t.parameters = [2]int{}
	t.parameterPresent = [2]bool{}
	t.parameterIndex = 0
}

func (t *terminal) writeEscape(value byte) {
	if t.escapeState == 1 {
		if value == '[' {
			t.escapeState = 2
			return
		}
		t.escapeState = 0
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
	t.finishCSI(value)
}

func (t *terminal) writeControl(value byte) {
	switch value {
	case 0x00:
		return
	case 0x07:
		t.bell = true
	case 0x08, 0x7f:
		t.backspace()
	case 0x09:
		t.tab()
	case 0x0a, 0x0b:
		t.lineFeed()
	case 0x0c:
		t.clear()
	case 0x0d:
		t.column = 0
	case 0x1b:
		t.escapeState = 1
	default:
		// With no child process to consume signals such as ETX, show their
		// conventional caret notation so Ctrl+letter remains observable.
		t.put('^')
		t.put(value + '@')
	}
}

func (t *terminal) WriteByte(value byte) {
	t.bell = false
	if t.escapeState != 0 {
		t.writeEscape(value)
		return
	}
	if value < 0x20 || value == 0x7f {
		t.writeControl(value)
		return
	}
	t.put(value)
}
