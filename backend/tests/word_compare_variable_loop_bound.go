package main

func sumBelow(bound int) int {
	total := 0
	for i := 0; i < bound; i++ {
		total += i
	}
	return total
}

func appMain() int {
	if sumBelow(3) != 3 || sumBelow(1) != 0 || sumBelow(0) != 0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
