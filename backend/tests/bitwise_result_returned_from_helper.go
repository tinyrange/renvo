package main

func bitmix(v int) int {
	return v ^ 0xff
}
func appMain(args []string) int {
	if !(bitmix(0x0f) == 0xf0) {
		print("RENVO-0208 bitwise_result_returned_from_helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
