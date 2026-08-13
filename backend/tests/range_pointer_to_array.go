package main

func pointerArrayTotal(values *[3]int) int {
	total := 0
	for index, value := range values {
		total += index + value
	}
	return total
}

func appMain() int {
	values := [3]int{7, 11, 24}
	if pointerArrayTotal(&values) != 45 {
		print("range over pointer-to-array failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
