package main

func shiftCount() int {
	return 2
}
func appMain(args []string) int {
	if !(4<<shiftCount() == 16) {
		print("RENVO-0234 shift_count_from_helper_return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
