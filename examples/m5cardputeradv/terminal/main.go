package main

import (
	board "renvo.dev/device/board/m5cardputeradv"
	"renvo.dev/std/graphics"
)

const (
	logicalWidth  = 120
	logicalHeight = 67
	cellWidth     = 6
	cellHeight    = 10
)

var (
	background  = graphics.RGBA(7, 12, 18, 255)
	foreground  = graphics.RGBA(205, 238, 214, 255)
	cursorColor = graphics.RGBA(65, 190, 120, 255)
	statusColor = graphics.RGBA(20, 50, 43, 255)
	alertColor  = graphics.RGBA(150, 45, 55, 255)
)

func keyName(key board.Key) string {
	switch key {
	case board.KeyEscape:
		return "ESC"
	case board.KeyF1:
		return "F1"
	case board.KeyF2:
		return "F2"
	case board.KeyF3:
		return "F3"
	case board.KeyF4:
		return "F4"
	case board.KeyF5:
		return "F5"
	case board.KeyF6:
		return "F6"
	case board.KeyF7:
		return "F7"
	case board.KeyF8:
		return "F8"
	case board.KeyF9:
		return "F9"
	case board.KeyF10:
		return "F10"
	case board.KeyF11:
		return "F11"
	case board.KeyF12:
		return "F12"
	case board.KeyBackspace:
		return "BACKSPACE"
	case board.KeyDelete:
		return "DELETE"
	case board.KeyTab:
		return "TAB"
	case board.KeyFunction:
		return "FN"
	case board.KeyShift:
		return "SHIFT"
	case board.KeyUp:
		return "UP"
	case board.KeyEnter:
		return "ENTER"
	case board.KeyControl:
		return "CTRL"
	case board.KeyOption:
		return "OPT"
	case board.KeyAlt:
		return "ALT"
	case board.KeyLeft:
		return "LEFT"
	case board.KeyDown:
		return "DOWN"
	case board.KeyRight:
		return "RIGHT"
	}
	return "KEY"
}

func appendText(buffer []byte, text string) []byte {
	for index := 0; index < len(text) && len(buffer) < terminalColumns; index++ {
		buffer = append(buffer, text[index])
	}
	return buffer
}

func statusFor(status *[terminalColumns]byte, event board.KeyEvent) {
	for index := 0; index < terminalColumns; index++ {
		status[index] = ' '
	}
	text := status[:0]
	text = appendText(text, "KEY ")
	if event.Character != 0 {
		text = append(text, event.Character)
	} else {
		text = appendText(text, keyName(event.Key))
	}
	if event.Pressed {
		text = appendText(text, " DOWN")
	} else {
		text = appendText(text, " UP")
	}
}

func initialStatus(status *[terminalColumns]byte) {
	for index := 0; index < terminalColumns; index++ {
		status[index] = ' '
	}
	copy(status[:], []byte("RENVO TERMINAL 20x6"))
}

func render(surface *graphics.Surface, font *graphics.Font, state *terminal, status *[terminalColumns]byte) {
	surface.FillRect(graphics.R(0, 0, logicalWidth, logicalHeight), background)
	for row := 0; row < terminalRows; row++ {
		for column := 0; column < terminalColumns; column++ {
			x := column * cellWidth
			y := row * cellHeight
			color := foreground
			if row == state.row && column == state.column {
				surface.FillRect(graphics.R(graphics.Scalar(x), graphics.Scalar(y), cellWidth, cellHeight), cursorColor)
				color = background
			}
			cell := state.cells[row][column : column+1]
			surface.DrawTextBytes(font, graphics.Point{X: graphics.Scalar(x), Y: graphics.Scalar(y + 7)}, cell, color)
		}
	}
	bar := statusColor
	if state.bell {
		bar = alertColor
	}
	surface.FillRect(graphics.R(0, 60, logicalWidth, 7), bar)
	surface.DrawTextBytes(font, graphics.Point{X: 0, Y: 66}, status[:], graphics.White)
}

func present(surface *graphics.Surface) {
	if board.Display.PresentSurface2x(surface) {
		surface.ResetDirty()
	}
}

func controlByte(character byte) byte {
	if character >= 'a' && character <= 'z' {
		return character - 'a' + 1
	}
	if character >= '@' && character <= '_' {
		return character & 0x1f
	}
	return character
}

func applyKey(state *terminal, event board.KeyEvent) {
	if !event.Pressed {
		return
	}
	if event.Character != 0 {
		value := event.Character
		if board.Keyboard.ControlPressed() {
			value = controlByte(value)
		}
		state.WriteByte(value)
		return
	}
	switch event.Key {
	case board.KeyBackspace, board.KeyDelete:
		state.WriteByte(0x08)
	case board.KeyTab:
		state.WriteByte(0x09)
	case board.KeyEnter:
		state.WriteByte(0x0d)
		state.WriteByte(0x0a)
	case board.KeyEscape:
		state.WriteByte(0x1b)
	case board.KeyLeft:
		state.move(0, -1)
	case board.KeyRight:
		state.move(0, 1)
	case board.KeyUp:
		state.move(-1, 0)
	case board.KeyDown:
		state.move(1, 0)
	}
}

func main() {
	if err := board.Keyboard.Initialize(); err != nil {
		for {
			print("FAIL: keyboard initialization\n")
			board.Clock.DelayMilliseconds(1000)
		}
	}

	font := graphics.NewBuiltinFont(1)
	surface := graphics.NewSurface(logicalWidth, logicalHeight)
	state := newTerminal()
	var status [terminalColumns]byte
	initialStatus(&status)
	render(surface, font, &state, &status)
	present(surface)
	print("PASS: Cardputer Adv terminal ready\n")

	for {
		event, ok, err := board.Keyboard.NextEvent()
		if err != nil {
			print("FAIL: keyboard transaction\n")
			board.Clock.DelayMilliseconds(100)
			continue
		}
		if !ok {
			board.Clock.DelayMilliseconds(2)
			continue
		}
		applyKey(&state, event)
		statusFor(&status, event)
		render(surface, font, &state, &status)
		present(surface)
	}
}
