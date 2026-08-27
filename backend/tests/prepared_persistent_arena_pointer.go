package main

type preparedArenaValue struct {
	number      byte
	initialized bool
}

func preparedArenaPointer(number byte) *preparedArenaValue {
	return &preparedArenaValue{number: number}
}

func appMain() int {
	value := preparedArenaPointer(25)
	if value == nil || value.number != 25 || value.initialized {
		return 1
	}
	print("PASS\n")
	return 0
}
