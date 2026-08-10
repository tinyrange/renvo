package main

import "renvo.dev/examples/m5nanoc6/board"

func mix(value uint32) uint32 {
	value = value ^ value<<13
	value = value ^ value>>17
	return value ^ value<<5
}

func main() {
	board.ConfigureButton()
	board.ConfigureRGB()
	board.SetRGB(0, 0, 0)

	timing := uint32(0)
	for {
		// Mix hardware RNG output with the exact human press timing. The timing
		// component also ensures successive samples do not depend solely on the
		// hardware generator's internal state.
		for !board.ButtonPressed() {
			timing++
		}
		board.DelayMilliseconds(10)
		if !board.ButtonPressed() {
			continue
		}

		random := mix(board.Random32() ^ timing)
		board.SetRGB(uint8(random>>16), uint8(random>>8), uint8(random))

		for board.ButtonPressed() {
			timing++
		}
		board.DelayMilliseconds(10)
	}
}
