package main

var renvoGlobalInitBase int = 5
var renvoGlobalInitExtra int = 7
var renvoGlobalInitTotal int = renvoGlobalInitBase + renvoGlobalInitExtra

func appMain(args []string) int {
	if renvoGlobalInitTotal != 12 {
		print("RENVO global var initializer from vars failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
