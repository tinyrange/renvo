package main

func appMain(args []string) int {
	var x byte = 'z'
	if !(x == 'z') {
		print("RENVO-0280 var_byte_explicit_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
