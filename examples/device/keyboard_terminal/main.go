package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/terminal"
	"renvo.dev/std/fmt"
	"renvo.dev/std/graphics"
)

func controlByte(character byte) byte {
	if character >= 'a' && character <= 'z' {
		return character - 'a' + 1
	}
	if character >= '@' && character <= '_' {
		return character & 0x1f
	}
	return character
}

func sendSequence(console *terminal.Terminal, text string) {
	for index := 0; index < len(text); index++ {
		console.SendInput(text[index])
	}
}

func applyKey(console *terminal.Terminal, event board.KeyEvent) {
	if !event.Pressed {
		return
	}
	if event.Character != 0 {
		value := event.Character
		if board.Keyboard.ControlPressed() {
			value = controlByte(value)
		}
		console.SendInput(value)
		return
	}
	switch event.Key {
	case board.KeyBackspace, board.KeyDelete:
		console.SendInput(0x08)
	case board.KeyTab:
		console.SendInput(0x09)
	case board.KeyEnter:
		console.SendInput('\r')
		console.SendInput('\n')
	case board.KeyEscape:
		console.SendInput(0x1b)
	case board.KeyLeft:
		sendSequence(console, "\x1b[D")
	case board.KeyRight:
		sendSequence(console, "\x1b[C")
	case board.KeyUp:
		sendSequence(console, "\x1b[A")
	case board.KeyDown:
		sendSequence(console, "\x1b[B")
	}
}

func main() {
	if err := board.Keyboard.Initialize(); err != nil {
		for {
			print("Cardputer terminal keyboard initialization failed: ", err.Error(), "\n")
			board.Clock.DelayMilliseconds(1000)
		}
	}

	console, err := terminal.Start(&board.Display, terminal.Options{
		Columns:     20,
		Rows:        6,
		Scrollback:  80,
		Font:        graphics.NewBuiltinFont(1),
		CellWidth:   6,
		CellHeight:  10,
		Baseline:    7,
		LocalEcho:   true,
		FlushPolicy: terminal.FlushEveryWrite,
		Clock:       &board.Clock,
	})
	if err != nil {
		for {
			print("Cardputer terminal display initialization failed: ", err.Error(), "\n")
			board.Clock.DelayMilliseconds(1000)
		}
	}

	fmt.Printf("\x1b[1;36mRENVO TERMINAL\x1b[0m\r\n")
	fmt.Printf("\x1b[32mstdout mirror ready\x1b[0m\r\n")
	fmt.Printf("type on the keyboard\r\n")

	for {
		event, ok, err := board.Keyboard.NextEvent()
		if err != nil {
			fmt.Printf("\x1b[31mkeyboard error: %s\x1b[0m\r\n", err)
			board.Clock.DelayMilliseconds(100)
			continue
		}
		if !ok {
			board.Clock.DelayMilliseconds(2)
			continue
		}
		applyKey(console, event)
		console.Flush()
	}
}
