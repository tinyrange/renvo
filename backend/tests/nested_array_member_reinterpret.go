package main

type nestedArrayTime struct {
	sec int64
}

type nestedArrayClock struct {
	base [1]nestedArrayTime
}

func nestedArrayRead(clock **nestedArrayClock) uint64 {
	return *(*uint64)(&((*clock)[0].base[0].sec))
}

func appMain(args []string) int {
	var clocks [1]nestedArrayClock
	clocks[0].base[0].sec = 42
	clock := &clocks[0]
	if nestedArrayRead(&clock) != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
