package main

func swapResults() (first int, second int) {
	defer func() {}()
	first = 4
	second = 9
	return second, first
}

func appMain(args []string) int {
	first, second := swapResults()
	if first != 9 || second != 4 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
