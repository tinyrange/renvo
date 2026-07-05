package main

var rtgGlobalInitBase int = 5
var rtgGlobalInitExtra int = 7
var rtgGlobalInitTotal int = rtgGlobalInitBase + rtgGlobalInitExtra

func appMain(args []string) int {
	if rtgGlobalInitTotal != 12 {
		print("RTG global var initializer from vars failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
