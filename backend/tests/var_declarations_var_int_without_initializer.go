package main

func appMain(args []string) int {
	var x int
	if !(x == 0) {
		print("RENVO-0276 var_int_without_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
