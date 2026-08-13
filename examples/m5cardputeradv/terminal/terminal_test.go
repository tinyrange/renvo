package main

import (
	"testing"

	board "renvo.dev/device/board/m5cardputeradv"
)

func line(t *terminal, row int) string { return string(t.cells[row][:]) }

func TestTerminalEditingControls(t *testing.T) {
	state := newTerminal()
	for _, value := range []byte("hello\b!\tX\rY") {
		state.WriteByte(value)
	}
	if got := line(&state, 0); got != "Yell!   X           " {
		t.Fatalf("line = %q", got)
	}
	state.WriteByte(0x0c)
	if line(&state, 0) != "                    " || state.row != 0 || state.column != 0 {
		t.Fatalf("form feed did not clear and home: row=%d column=%d line=%q", state.row, state.column, line(&state, 0))
	}
}

func TestTerminalScrolls(t *testing.T) {
	state := newTerminal()
	for row := 0; row < terminalRows+1; row++ {
		state.WriteByte(byte('0' + row))
		state.WriteByte('\r')
		state.WriteByte('\n')
	}
	if state.cells[terminalRows-2][0] != '6' || state.row != terminalRows-1 {
		t.Fatalf("scroll bottom = %q, row = %d", line(&state, terminalRows-2), state.row)
	}
}

func TestTerminalCaretAndANSIControls(t *testing.T) {
	state := newTerminal()
	state.WriteByte(0x03)
	if line(&state, 0)[:2] != "^C" {
		t.Fatalf("ETX = %q", line(&state, 0))
	}
	for _, value := range []byte("abc\x1b[2D!") {
		state.WriteByte(value)
	}
	if line(&state, 0)[:5] != "^Ca!c" {
		t.Fatalf("cursor-left sequence = %q", line(&state, 0))
	}
	for _, value := range []byte("\x1b[2J") {
		state.WriteByte(value)
	}
	if state.row != 0 || state.column != 0 || line(&state, 0) != "                    " {
		t.Fatalf("erase display failed: row=%d column=%d line=%q", state.row, state.column, line(&state, 0))
	}
	for _, value := range []byte("\x1b[3;5H") {
		state.WriteByte(value)
	}
	if state.row != 2 || state.column != 4 {
		t.Fatalf("cursor position = %d,%d", state.row, state.column)
	}
}

func TestKeyIdentificationAndControlConversion(t *testing.T) {
	var status [terminalColumns]byte
	statusFor(&status, board.KeyEvent{Key: 'a', Character: 'A', Pressed: true})
	if got := string(status[:]); got != "KEY A DOWN          " {
		t.Fatalf("status = %q", got)
	}
	statusFor(&status, board.KeyEvent{Key: board.KeyBackspace, Pressed: true})
	if got := string(status[:]); got != "KEY BACKSPACE DOWN  " {
		t.Fatalf("special status = %q", got)
	}
	if controlByte('c') != 0x03 || controlByte('[') != 0x1b {
		t.Fatal("control conversion mismatch")
	}
}
