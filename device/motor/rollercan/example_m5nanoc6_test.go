//go:build m5nanoc6

package rollercan_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/motor/rollercan"
)

func ExampleDevice_Speed() {
	motor := rollercan.New(i2c.New(board.Grove))
	speed, err := motor.Speed()
	if err == nil {
		_ = speed /* Hundredths of RPM. */
	}
}
