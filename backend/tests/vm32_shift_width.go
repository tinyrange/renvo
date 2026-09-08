package main

func checkShift(value int32, count uint32) bool {
	return value<<count == 0 && value>>count == 0 && uint32(value)>>count == 0 && -value>>count == -1
}

func appMain() int {
	if !checkShift(8, 32) || !checkShift(8, 48) {
		return 1
	}
	print("PASS\n")
	return 0
}
