package main

import "renvo.dev/device/board"

// These functions are the intentionally small board API presented to C.
func board_set_led(on int32) {
	board.LED.Set(on != 0)
}

func board_delay_milliseconds(milliseconds uint32) {
	board.Clock.DelayMilliseconds(milliseconds)
}
