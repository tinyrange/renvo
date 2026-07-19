package main

func appMain(args []string) int {
	var x = 13
	if !(x == 13) {
		print("RENVO-0278 var_int_inferred_from_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
