package main

func callArgumentMark(trace *int, digit int) int {
	*trace = *trace*10 + digit
	return digit
}

func callArgumentFirst(left int, right int) int {
	return left
}

func appMain() int {
	trace := 0
	if callArgumentFirst(callArgumentMark(&trace, 1), callArgumentMark(&trace, 2)) != 1 || trace != 12 {
		print("FAIL: call argument evaluation order\n")
		return 1
	}
	print("PASS\n")
	return 0
}
