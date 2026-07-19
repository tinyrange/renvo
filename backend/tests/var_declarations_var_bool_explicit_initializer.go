package main

func appMain(args []string) int {
	var x bool = true
	if !(x) {
		print("RENVO-0281 var_bool_explicit_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
