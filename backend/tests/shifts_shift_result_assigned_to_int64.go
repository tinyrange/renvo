package main

func appMain(args []string) int {
	var x int64 = 3 << 5
	if !(x == 96) {
		print("RENVO-0241 shift_result_assigned_to_int64 failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
