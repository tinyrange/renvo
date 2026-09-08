package main

func helperTailAllocation(size int) byte {
	data := make([]byte, size)
	data[size-1] = 1
	return data[size-1]
}

func appMain() int {
	if helperTailAllocation(4097) != 1 {
		return 1
	}
	print("PASS\n")
	return 0
}
