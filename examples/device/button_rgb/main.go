package main

import "renvo.dev/device/board"

func mix(value uint32) uint32 {
	value = value ^ value<<13
	value = value ^ value>>17
	return value ^ value<<5
}

func main() {
	board.RGB.Set(0, 0, 0)

	timing := uint32(0)
	for {
		// Mix hardware RNG output with the exact human press timing. The timing
		// component also ensures successive samples do not depend solely on the
		// hardware generator's internal state.
		for !board.Button.Pressed() {
			timing++
		}
		board.Clock.DelayMilliseconds(10)
		if !board.Button.Pressed() {
			continue
		}

		random := mix(board.Random.Uint32() ^ timing)
		board.RGB.Set(uint8(random>>16), uint8(random>>8), uint8(random))

		for board.Button.Pressed() {
			timing++
		}
		board.Clock.DelayMilliseconds(10)
	}
}
