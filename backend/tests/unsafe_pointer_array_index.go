package main

func storeUnsafeArrayWord(address *[16]uint32, index int, value uint32) {
	address[index] = value
}

func appMain(args []string) int {
	var words [16]uint32
	storeUnsafeArrayWord(&words, 3, 42)
	if words[3] != 42 {
		print("FAIL: unsafe pointer array index\n")
		return 1
	}
	print("PASS\n")
	return 0
}
