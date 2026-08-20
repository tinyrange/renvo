package main

var globalNarrowCount int32
var globalNarrowNeighbor uint32 = 12

func appMain(args []string) int {
	if globalNarrowCount != 0 || globalNarrowNeighbor != 12 {
		return 1
	}
	print("PASS\n")
	return 0
}
