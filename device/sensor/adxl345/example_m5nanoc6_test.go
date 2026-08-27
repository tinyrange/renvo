//go:build m5nanoc6

package adxl345_test

import (
	"renvo.dev/device/board"
	"renvo.dev/device/i2c"
	"renvo.dev/device/sensor/adxl345"
)

func ExampleDevice_Read() {
	sensor := adxl345.New(i2c.New(board.Grove), adxl345.AddressLow)
	reading, err := sensor.Read()
	if err == nil {
		_, _, _ = reading.X, reading.Y, reading.Z
	}
}
