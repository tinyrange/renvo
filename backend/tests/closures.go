package main

func closureMakeAdder(base int) func(int) int {
	return func(value int) int {
		return base + value
	}
}

func closureMakeCounter(start int) func() int {
	count := start
	return func() int {
		count = count + 1
		return count
	}
}

func appMain() int {
	add := closureMakeAdder(4)
	if add(3) != 7 {
		return 1
	}
	next := closureMakeCounter(10)
	if next() != 11 || next() != 12 {
		return 1
	}
	print("PASS\n")
	return 0
}
