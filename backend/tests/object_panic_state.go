package main

func objectPanicCleanup(value *uint32) { *value = *value + 1 }

//export ObjectPanicState
func objectPanicState(value *uint32) uint32 {
	defer objectPanicCleanup(value)
	return 42
}

func appMain() int {
	var value uint32
	if objectPanicState(&value) != 42 || value != 1 {
		return 1
	}
	print("PASS\n")
	return 0
}
