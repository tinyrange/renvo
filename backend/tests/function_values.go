package main

type intBinary func(int, int) int

func functionValueAdd(left int, right int) int {
	return left + right
}

func functionValueMultiply(left int, right int) int {
	return left * right
}

var functionValueGlobal = functionValueAdd

func functionValueMark(trace *int, digit int) int {
	*trace = *trace*10 + digit
	return digit
}

func functionValueApply(fn func(int, int) int, left int, right int) int {
	return fn(left, right)
}

func appMain() int {
	trace := 0
	var fn intBinary = functionValueAdd
	left := functionValueMark(&trace, 2)
	right := functionValueMark(&trace, 3)
	if functionValueApply(fn, left, right) != 5 || trace != 23 {
		return 1
	}
	if functionValueGlobal(4, 5) != 9 {
		return 1
	}
	fn = functionValueMultiply
	if fn(3, 4) != 12 {
		return 1
	}
	print("PASS\n")
	return 0
}
