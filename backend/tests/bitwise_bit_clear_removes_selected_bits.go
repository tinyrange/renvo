package main

func appMain(args []string) int {
	if !(0x3f&^0x0f == 0x30) {
		print("RENVO-0204 bit_clear_removes_selected_bits failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
