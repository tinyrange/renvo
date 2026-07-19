package main

func appMain(args []string) int {
	b := byte(1 << 6)
	if !(int(b) == 64) {
		print("RENVO-0242 shift_result_converted_to_byte failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
