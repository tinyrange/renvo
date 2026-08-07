package main

type capturedAddressValue struct {
	numbers [2]int
}

func appMain(args []string) int {
	state := capturedAddressValue{}
	pointer := &state
	apply := func(delta int) capturedAddressValue {
		pointer.numbers[0] += delta
		return state
	}
	first := apply(0)
	second := apply(1)
	if first.numbers[0]+second.numbers[0] != 1 {
		print("FAIL: captured address did not alias captured cell\n")
		return 1
	}
	print("PASS\n")
	return 0
}
