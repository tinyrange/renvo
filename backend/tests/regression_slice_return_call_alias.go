package main

func returnInputSlice(values []int) []int {
	return values
}

func returnCallSlice(values []int) []int {
	return returnInputSlice(values)
}

func buildLocalViaCall(value int) []int {
	values := make([]int, 0, 1)
	values = append(values, value)
	return returnInputSlice(values)
}

func appendParam(values []int, value int) []int {
	return append(values, value)
}

func appMain(args []string) int {
	values := make([]int, 0, 4)
	values = append(values, 3)
	alias := returnCallSlice(values)

	values[0] = 7
	if alias[0] != 7 {
		print("FAIL\n")
		return 1
	}

	alias[0] = 11
	if values[0] != 11 {
		print("FAIL\n")
		return 1
	}

	appended := appendParam(values, 13)
	appended[0] = 17
	if values[0] != 17 {
		print("FAIL\n")
		return 1
	}
	if appended[1] != 13 {
		print("FAIL\n")
		return 1
	}

	escaped := buildLocalViaCall(21)
	buildLocalViaCall(31)
	buildLocalViaCall(41)
	buildLocalViaCall(51)
	if escaped[0] != 21 {
		print("FAIL\n")
		return 1
	}

	print("PASS\n")
	return 0
}
