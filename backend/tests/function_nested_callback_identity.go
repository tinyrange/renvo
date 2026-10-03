package main

func nestedCallbackApply(f func(value int) int) int { return f(42) }

func appMain() int {
	var call func(func(int) int) int = nestedCallbackApply
	if call(func(value int) int { return value }) != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
