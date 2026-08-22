package main

func wideUnsignedBool(value bool) int {
	if value {
		return 1
	}
	return 0
}

func wideUnsignedCompare(left int, right int) (int, bool) {
	return wideUnsignedBool(uint64(left) <= uint64(right)), true
}

func appMain(args []string) int {
	value, ok := wideUnsignedCompare(41, 42)
	if !ok || value != 1 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
