package main

func writeNarrowNegative(value *int32) {
	*value = -1
}

func appMain(args []string) int {
	value := int32(0)
	writeNarrowNegative(&value)
	if value >= 0 {
		return 1
	}
	print("PASS\n")
	return 0
}
