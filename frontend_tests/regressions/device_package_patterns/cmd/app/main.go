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
	read := []byte{0}
	err := bus.Tx(0x58, []byte{1, 2}, read)
	if err != nil || read[0] != 9 {
		print("FAIL: port transaction\n")
		return
	}
	err = bus.Write(0x58, []byte{3})
	if err != nil {
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
