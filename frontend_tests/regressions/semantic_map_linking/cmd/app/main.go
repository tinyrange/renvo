package main

var trace int

func mapKey(step int) string {
	trace = trace*10 + step
	if step == 1 {
		return "alpha"
	}
	if step == 3 {
		return "beta"
	}
	return "gamma"
}

func mapValue(step int, value int) int {
	trace = trace*10 + step
	return value
}

func main() {
	values := map[string]int{
		mapKey(1): mapValue(2, 4),
		mapKey(3): mapValue(4, 5),
		mapKey(5): mapValue(6, 6),
	}
	values[mapKey(3)] = values[mapKey(1)] + values[mapKey(5)]
	if trace == 123456315 && values["alpha"] == 4 && values["beta"] == 10 && values["gamma"] == 6 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
