package main

func returnedSliceWithExcessCapacity() ([]byte, bool) {
	value := make([]byte, 4, 256)
	value[0] = 1
	return value, true
}

func appMain() int {
	value, ok := returnedSliceWithExcessCapacity()
	if !ok || len(value) != 4 || cap(value)-len(value) != 80 || value[0] != 1 {
		return 1
	}
	print("PASS\n")
	return 0
}
