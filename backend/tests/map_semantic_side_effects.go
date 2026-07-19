package main

var semanticMapTrace int

func semanticMapKey(step int) string {
	semanticMapTrace = semanticMapTrace*10 + step
	if step == 1 {
		return "alpha"
	}
	if step == 3 {
		return "beta"
	}
	return "gamma"
}

func semanticMapValue(step int, value int) int {
	semanticMapTrace = semanticMapTrace*10 + step
	return value
}

func appMain(args []string) int {
	values := map[string]int{
		semanticMapKey(1): semanticMapValue(2, 4),
		semanticMapKey(3): semanticMapValue(4, 5),
		semanticMapKey(5): semanticMapValue(6, 6),
	}
	values[semanticMapKey(3)] = values[semanticMapKey(1)] + values[semanticMapKey(5)]
	if semanticMapTrace == 123456315 && values["alpha"] == 4 && values["beta"] == 10 && values["gamma"] == 6 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
