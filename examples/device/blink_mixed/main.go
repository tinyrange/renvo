package main

import "renvo.dev/device/board"

// These narrow functions are the Go side of the shared C/Go boundary. The C
// implementation can stay board-agnostic while Go supplies Renvo's typed board
// capabilities.
func goSetBlueLED(on int) {
	board.BlueLED.Set(on != 0)
}

func goDelayMilliseconds(milliseconds int) {
	board.Clock.DelayMilliseconds(uint32(milliseconds))
}

func main() {
	// cBlinkForever is implemented in blink.c and calls back into the two Go
	// functions above for GPIO and timing.
	cBlinkForever(500)
}
