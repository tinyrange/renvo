package main

func appMain() int {
	values := []int{3, 4}
	position := 0
	position, values[position] = 1, 7
	if position != 1 || values[0] != 7 || values[1] != 4 {
		print("FAIL: multi-assignment index evaluation order\n")
		return 1
	}
	print("PASS\n")
	return 0
}
