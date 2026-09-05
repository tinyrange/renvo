package main

const complemented = ^uint8(0)

func appMain(args []string) int {
	if ^uint8(0) != 255 || complemented != 255 || ^uint16(0) != 65535 || ^int8(0) != -1 { return 1 }
	print("PASS\n")
	return 0
}
