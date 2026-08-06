package main

func threeArgumentMark(trace *int, digit int) int {
	*trace = *trace*10 + digit
	return digit
}

func threeArgumentCombine(first int, second int, third int) int {
	return first*100 + second*10 + third
}

func appMain() int {
	trace := 0
	result := threeArgumentCombine(threeArgumentMark(&trace, 1), threeArgumentMark(&trace, 2), threeArgumentMark(&trace, 3))
	if result != 123 || trace != 123 {
		print("FAIL: three argument evaluation order\n")
		return 1
	}
	print("PASS\n")
	return 0
}
