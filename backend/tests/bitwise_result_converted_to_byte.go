package main

func appMain(args []string) int {
	b := byte(0x123 & 0xff)
	if !(int(b) == 35) {
		print("RENVO-0220 bitwise_result_converted_to_byte failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
