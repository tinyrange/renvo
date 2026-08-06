package main

func nestedCallArgumentMark(trace *int, digit int) int {
	*trace = *trace*10 + digit
	return digit
}

func nestedCallArgumentFirst(left int, right int) int {
	return left
}

func appMain() int {
	trace := 0
	result := nestedCallArgumentFirst(nestedCallArgumentMark(&trace, 1)+1, nestedCallArgumentMark(&trace, 2)*2)
	if result != 2 || trace != 12 {
		print("FAIL: nested call argument evaluation order\n")
		return 1
	}
	print("PASS\n")
	return 0
}
