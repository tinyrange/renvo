//go:build m5tab5

package tab5keyboard_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/input/tab5keyboard"
)

// NextCharacter reads a translated key press from the Tab5 Keyboard on Ext.Port1.
func ExampleDevice_NextCharacter() {
	keyboard := tab5keyboard.New(i2c.New(board.ExtPort1()))
	if err := keyboard.Initialize(tab5keyboard.Character); err != nil {
		return
	}
	event, ok, err := keyboard.NextCharacter()
	if err == nil && ok {
		var input [16]byte
		n := event.TerminalBytes(input[:])
		print(string(input[:n]))
	}
}
