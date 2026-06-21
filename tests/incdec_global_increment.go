package main

var rtgIncdecGlobalValue int = 8

func appMain(args []string) int {
	rtgIncdecGlobalValue++
	if rtgIncdecGlobalValue != 9 {
		print("RTG-INCDEC-007 global increment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
