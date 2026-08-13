package main

import "renvo.dev/examples/m5tab5/board"

func main() {
	print("TAB5 TOUCH PROBE START\n")
	if !board.InitPower() {
		failure := board.PowerFailure()
		if failure == 1 {
			print("TAB5 POWER RESET WRITE FAIL\n")
		} else if failure == 2 {
			print("TAB5 POWER RESET READ FAIL\n")
		} else if failure == 3 {
			print("TAB5 POWER DIRECTION FAIL\n")
		} else if failure == 4 {
			print("TAB5 POWER IMPEDANCE FAIL\n")
		} else if failure == 5 {
			print("TAB5 POWER PULL SELECT FAIL\n")
		} else if failure == 6 {
			print("TAB5 POWER PULL ENABLE FAIL\n")
		} else if failure == 7 {
			print("TAB5 POWER OUTPUT FAIL\n")
		}
		print("TAB5 POWER FAIL\n")
		for {
		}
	}
	print("TAB5 POWER PASS\n")
	// ST7121 integrates the display and touch controllers. After a cold reset,
	// its I2C touch endpoint does not respond until the DSI display side is up.
	if !board.InitFramebuffer() {
		print("TAB5 DISPLAY FAIL\n")
		for {
		}
	}
	print("TAB5 DISPLAY PASS\n")
	if !board.InitTouch() {
		print("TAB5 TOUCH FAIL\n")
		for {
		}
	}
	print("TAB5 TOUCH PASS\n")
	for {
	}
}
