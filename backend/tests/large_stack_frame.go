package main

func largeStackFrame() int {
	var data [9000]int
	data[0] = 42
	data[8999] = 43
	return data[0] + data[8999]
}

func appMain() int {
	if largeStackFrame() != 85 {
		panic("large frame")
	}
	print("PASS\n")
	return 0
}
