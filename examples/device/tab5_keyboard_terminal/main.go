package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/input/tab5keyboard"
	"renvo.dev/device/terminal"
	"renvo.dev/std/fmt"
	"renvo.dev/std/graphics"
)

func main() {
	print("RENVO TAB5 KEYBOARD boot\n")
	board.Display.SetLandscape(true)
	console, err := terminal.Start(&board.Display, terminal.Options{
		Scrollback: 128, Font: graphics.NewBuiltinFont(2),
		CellWidth: 12, CellHeight: 20, Baseline: 14,
		LocalEcho: true, FlushPolicy: terminal.FlushManual, Clock: &board.Display,
		Pointer: &board.Touch,
	})
	if err != nil {
		print("terminal: ", err.Error(), "\n")
		return
	}
	keyboard := tab5keyboard.New(i2c.New(board.ExtPort1()))
	// This is a raw VT terminal: LF moves down, CR returns to column zero.
	// Standard output is mirrored here too, so all displayed lines need CRLF.
	print("Tab5 display ready: landscape native RGB565\r\n")
	fmt.Fprint(console, "\x1b[36mRENVO TAB5 TERMINAL\x1b[0m\r\n")
	fmt.Fprint(console, "Ctrl+O: portrait / landscape. Ctrl+L: clear. Arrow keys move the cursor.\r\n")
	fmt.Fprint(console, "Local echo demo; no shell or remote connection. Touch-drag for scrollback.\r\n")
	console.Flush()
	for {
		if err = keyboard.Initialize(tab5keyboard.Character); err == nil {
			break
		}
		fmt.Fprintf(console, "Keyboard unavailable: %s\r\nAttach to Ext.Port1; retrying...\r\n", err)
		console.Flush()
		board.Display.DelayMilliseconds(1000)
	}
	version, _ := keyboard.FirmwareVersion()
	fmt.Printf("Tab5 Keyboard ready, firmware %d\r\n", version)
	var input [16]byte
	for console.Tick(16 * terminal.Millisecond) {
		// Drain a bounded batch so display servicing cannot be starved by input.
		for pending := 0; pending < 32; pending++ {
			event, ok, readErr := keyboard.NextCharacter()
			if readErr != nil {
				fmt.Fprintf(console, "\r\nKeyboard: %s\r\n", readErr)
				board.Display.DelayMilliseconds(100)
				break
			}
			if !ok {
				break
			}
			n := event.TerminalBytes(input[:])
			if n == 1 && input[0] == 15 {
				board.Display.SetLandscape(!board.Display.Landscape)
				if !console.Flush() {
					return
				}
				fmt.Printf("\r\nOrientation: landscape=%t, grid=%dx%d\r\n", board.Display.Landscape, console.Columns(), console.Rows())
			} else if n == 1 && input[0] == 12 {
				console.Reset()
			} else {
				for i := 0; i < n; i++ {
					console.SendInput(input[i])
				}
			}
			// Local echo consumes display output but still queues input for a
			// future shell/transport. This demo discards that queue deliberately.
			console.Read(input[:])
		}
	}
}
