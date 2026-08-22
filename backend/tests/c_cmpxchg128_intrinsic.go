package main

func renvo_runtime_CCompareExchange128(address *[2]uint64, oldLow *uint64, oldHigh *uint64, newLow uint64, newHigh uint64) uint8 {
	if address[0] == *oldLow && address[1] == *oldHigh {
		address[0] = newLow
		address[1] = newHigh
		return 1
	}
	*oldLow = address[0]
	*oldHigh = address[1]
	return 0
}

var compareExchangeValue [2]uint64

func appMain(args []string) int {
	value := &compareExchangeValue
	value[0], value[1] = 3, 5
	oldLow, oldHigh := uint64(3), uint64(5)
	if renvo_runtime_CCompareExchange128(value, &oldLow, &oldHigh, 7, 11) != 1 ||
		value[0] != 7 || value[1] != 11 || oldLow != 3 || oldHigh != 5 {
		return 1
	}
	oldLow, oldHigh = 13, 17
	if renvo_runtime_CCompareExchange128(value, &oldLow, &oldHigh, 19, 23) != 0 ||
		value[0] != 7 || value[1] != 11 || oldLow != 7 || oldHigh != 11 {
		return 2
	}
	print("PASS\n")
	return 0
}
