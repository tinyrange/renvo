package main

var calls int

func source() []int {
	calls++
	return []int{3}
}

func shadow(nil []int) []int {
	values := []int{1}
	return append(values, nil...)
}

func appMain(args []string) int {
	values := []int{1, 2}
	values = append(values, nil...)
	if len(values) != 2 || values[0] != 1 || values[1] != 2 {
		print("FAIL\n")
		return 1
	}
	var result []int
	result = append(source(), nil...)
	if calls != 1 || len(result) != 1 || result[0] != 3 {
		print("FAIL\n")
		return 1
	}
	short := append(source(), nil...)
	if calls != 2 || len(short) != 1 || short[0] != 3 {
		print("FAIL\n")
		return 1
	}
	result = shadow([]int{2})
	if len(result) != 2 || result[1] != 2 {
		print("FAIL\n")
		return 1
	}
	result = append(result)
	if len(result) != 2 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
