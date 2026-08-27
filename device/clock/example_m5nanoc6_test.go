//go:build m5nanoc6

package clock_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/clock"
	"renvo.dev/device/esp32c6"
)

// Milliseconds provides wrap-safe elapsed time from the board clock.
func ExampleClock_Milliseconds() {
	started := board.Clock.Milliseconds()
	board.Clock.DelayMilliseconds(100)
	elapsed := board.Clock.Milliseconds() - started
	_ = elapsed
}

func ExampleNew() {
	source := esp32c6.SystemTimer{}
	timer := clock.New(source)
	_ = timer.Milliseconds()
}

func ExampleClock_DelayMicroseconds() {
	board.Clock.DelayMicroseconds(10) // Peripheral power-up settling time.
}

func ExampleClock_DelayMilliseconds() {
	board.Clock.DelayMilliseconds(100)
}

// DelayUntil keeps a polling loop on schedule even when the work duration varies.
func ExampleClock_DelayUntil() {
	started := board.Clock.Ticks()
	for attempt := 0; attempt < 10; attempt++ {
		// Poll the peripheral.
		board.Clock.DelayUntil(started, uint32(attempt+1)*1_000)
	}
}

func ExampleClock_Ticks() {
	started := board.Clock.Ticks()
	// Perform a short hardware operation.
	elapsedTicks := board.Clock.Ticks() - started
	_ = elapsedTicks
}
