//go:build m5nanoc6

package tca8418_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/input/tca8418"
)

// NextEvent removes one physical key transition from the controller FIFO.
func ExampleDevice_NextEvent() {
	keypad := tca8418.New(i2c.New(board.Grove))
	event, ok, err := keypad.NextEvent()
	if err == nil && ok {
		_, _, _ = event.Row, event.Column, event.Pressed
	}
}
