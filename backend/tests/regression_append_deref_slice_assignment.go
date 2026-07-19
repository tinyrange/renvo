package main

func appMain() int {
	values := []int{}
	ptr := &values
	*ptr = append(*ptr, 3)
	if (*ptr)[0] == 3 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
