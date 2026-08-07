package main

func appMain(args []string) int {
	var values []int64
	values = append(values, int64(0x100000002), int64(-3))
	if len(values) != 2 || values[0] != int64(0x100000002) || values[1] != int64(-3) {
		print("FAIL: append wide scalar\n")
		return 1
	}
	print("PASS\n")
	return 0
}
