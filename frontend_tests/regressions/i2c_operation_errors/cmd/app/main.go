package main

import (
	"fmt"

	"renvo.dev/device/i2c"
)

type controller struct{}

func (*controller) Configure() error                { return nil }
func (*controller) Tx(uint16, []byte, []byte) error { return i2c.ErrNAK }

type delay struct{}

func (*delay) DelayMicroseconds(uint32) {}
func (*delay) DelayMilliseconds(uint32) {}

func main() {
	bus := i2c.New(i2c.DefinePort(&controller{}, &delay{}))
	_, err := bus.Device(0x53).ReadAt(make([]byte, 1), 0x00)
	if fmt.Sprint(err) != "i2c read at address 0x53, register 0x00: target did not acknowledge the I2C byte" {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
