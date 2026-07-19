package main

var globalVarNoInit int

func appMain(args []string) int {
	if !(globalVarNoInit == 0) {
		print("RENVO-0298 global_var_without_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
