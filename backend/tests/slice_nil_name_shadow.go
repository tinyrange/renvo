package main

func length(nil []int) int {
	value := nil
	return len(value)
}

func appMain(args []string) int {
	if length([]int{1, 2}) != 2 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
