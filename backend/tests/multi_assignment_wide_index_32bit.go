package main

func appMain(args []string) int {
	values := []int64{-1, 0, 0}
	position := 0
	position, values[position] = 0, 0
	if position != 0 || values[0] != 0 {
		print("FAIL: multi-assignment wide index\n")
		return 1
	}
	print("PASS\n")
	return 0
}
