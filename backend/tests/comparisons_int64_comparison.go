package main

func appMain(args []string) int {
	var a int64 = 44
	var b int64 = 45
	if !(a < b) {
		print("RENVO-0199 int64_comparison failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
