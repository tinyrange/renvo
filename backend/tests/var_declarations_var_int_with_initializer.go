package main

func appMain(args []string) int {
	var x int = 12
	if !(x == 12) {
		print("RENVO-0277 var_int_with_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
