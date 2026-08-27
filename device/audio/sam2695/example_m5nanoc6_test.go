//go:build m5nanoc6

package sam2695_test

import (
	"renvo.dev/device/audio/sam2695"
	"renvo.dev/device/board"
	"renvo.dev/device/uart"
)

// A SAM2695 uses the MIDI rate of 31,250 baud.
func ExampleDevice_NoteOn() {
	serial := uart.New(board.GroveUART, 31_250)
	synth := sam2695.New(serial)
	_ = synth.NoteOn(0, 60, 96) // Middle C on MIDI channel 1.
}
