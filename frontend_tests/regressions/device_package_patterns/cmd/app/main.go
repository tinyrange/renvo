package main

import (
	board "example.com/renvotests/regressions/device_package_patterns/board"
	"renvo.dev/device/i2c"
)

func main() {
	led, pinConfigurations, busConfigurations, _, _, _ := board.State()
	if led || pinConfigurations != 0 || busConfigurations != 0 {
		print("FAIL: eager initialization\n")
		return
	}

	board.BlueLED.Set(true)
	board.PressButton()
	if !board.Button.Pressed() {
		print("FAIL: package-global device\n")
		return
	}
	board.Clock.DelayMicroseconds(3)

	bus := i2c.New(board.Grove)
	device := bus.Device(0x58)
	read := []byte{0}
	n, err := device.ReadAt(read, 1)
	if err != nil || n != 1 || read[0] != 9 {
		print("FAIL: port transaction\n")
		return
	}
	n, err = device.WriteAt([]byte{3}, 2)
	if err != nil || n != 1 {
		print("FAIL: repeated transaction\n")
		return
	}

	led, pinConfigurations, busConfigurations, address, writes, reads := board.State()
	if !led || pinConfigurations != 1 || busConfigurations != 1 ||
		address != 0x58 || writes != 3 || reads != 1 {
		print("FAIL: composed state\n")
		return
	}
	print("PASS\n")
}
