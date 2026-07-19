package main

var renvoIncdecGlobalValue int = 8

func appMain(args []string) int {
	renvoIncdecGlobalValue++
	if renvoIncdecGlobalValue != 9 {
		print("RENVO-INCDEC-007 global increment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
