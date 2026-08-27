//go:build m5nanoc6

package miniscale_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/miniscale"
)

// The fixed-point reading avoids float rounding in display and control loops.
func ExampleDevice_ReadWeightHundredths() {
	scale := miniscale.New(i2c.New(board.Grove))
	weight, err := scale.ReadWeightHundredths()
	if err == nil {
		_ = weight // Hundredths of a gram.
	}
}
