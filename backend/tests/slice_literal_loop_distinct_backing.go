package main

func appMain() int {
	values := make([][]int, 3)
	for i := 0; i < len(values); i++ {
		values[i] = []int{i}
	}
	if values[0][0] == 0 && values[1][0] == 1 && values[2][0] == 2 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
