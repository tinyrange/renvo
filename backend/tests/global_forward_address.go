package main

var globalForwardPointer = &globalForwardTarget
var globalForwardTarget = 42

func appMain() int {
	if globalForwardPointer == nil || *globalForwardPointer != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
