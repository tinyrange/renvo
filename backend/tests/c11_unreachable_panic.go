package main

// renvo:c11

func guardedCallback(kind int) int {
	if kind == 1 {
		return 42
	}
	panic("call of nil function")
	return 0
}

func appMain() int {
	if guardedCallback(1) != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
