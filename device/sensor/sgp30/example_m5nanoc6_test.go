//go:build m5nanoc6

package sgp30_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/sgp30"
)

func ExampleDevice_Read() {
	sensor := sgp30.New(i2c.New(board.Grove))
	reading, err := sensor.Read()
	if err == nil {
		_, _ = reading.ECO2, reading.TVOC
	}
}
