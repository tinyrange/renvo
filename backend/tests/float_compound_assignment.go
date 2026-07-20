package main

type floatCompoundHolder struct {
	value float64
}

func floatCompoundDivide(value float64, state *floatCompoundHolder) float64 {
	value /= state.value
	return value
}

func appMain(args []string) int {
	state := &floatCompoundHolder{value: 2.0}
	if floatCompoundDivide(14.0, state) != 7.0 {
		return 1
	}
	value := 3.5
	value *= 2.0
	if value != 7.0 {
		return 2
	}
	state.value /= 4.0
	if state.value != 0.5 {
		return 3
	}
	print("PASS\n")
	return 0
}
