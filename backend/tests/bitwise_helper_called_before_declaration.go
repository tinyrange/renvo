package main

func bitmix(v int) int {
	return v ^ 0xff
}
func appMain(args []string) int {
	if !(bitmix(0x33) == 0xcc) {
		print("RENVO-0222 bitwise_helper_called_before_declaration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
