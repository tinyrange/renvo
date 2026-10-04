package main

func compoundLocalMutate(value *int) int {
	*value = 99
	return 4
}

func appMain() int {
	value := 7
	value += compoundLocalMutate(&value)
	if value != 11 {
		return 1
	}
	print("PASS\n")
	return 0
}
