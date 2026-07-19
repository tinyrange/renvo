package main

func appMain(args []string) int {
	var x int64 = 14
	if !(x == 14) {
		print("RENVO-0279 var_int64_explicit_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
