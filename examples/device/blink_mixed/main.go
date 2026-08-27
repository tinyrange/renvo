package main

import "renvo.dev/device/board"

/* void cBlinkForever(int intervalMilliseconds); */
import "C"

// These narrow functions are the explicitly exported Go side of the C/Go
// boundary. The C implementation can stay board-agnostic while Go supplies
// Renvo's typed board capabilities.
//
//export goSetBlueLED
func goSetBlueLED(on int) {
	board.BlueLED.Set(on != 0)
}

//export goDelayMilliseconds
func goDelayMilliseconds(milliseconds int) {
	board.Clock.DelayMilliseconds(uint32(milliseconds))
}

func main() {
	// C.cBlinkForever is implemented in blink.c and calls back into the two Go
	// functions above for GPIO and timing.
	C.cBlinkForever(500)
}
