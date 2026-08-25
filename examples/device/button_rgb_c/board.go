package main

import "renvo.dev/device/board"

func board_button_pressed() int32 {
	if board.Button.Pressed() {
		return 1
	}
	return 0
}

func board_random_uint32() uint32 {
	return board.Random.Uint32()
}

func board_set_rgb(red uint8, green uint8, blue uint8) {
	board.RGB.Set(red, green, blue)
}

func board_delay_milliseconds(milliseconds uint32) {
	board.Clock.DelayMilliseconds(milliseconds)
}
