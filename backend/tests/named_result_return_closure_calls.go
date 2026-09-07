package main

func closureReturnValues() (first int, second int) {
	first, second = 4, 9
	defer func() { first++ }()
	return func() int { first = 8; return second }(), func() int { return first }()
}

func appMain(args []string) int {
	first, second := closureReturnValues()
	if first != 10 || second != 8 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
