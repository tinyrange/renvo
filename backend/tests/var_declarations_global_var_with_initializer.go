package main

var globalVarInit int = 42

func appMain(args []string) int {
	if !(globalVarInit == 42) {
		print("RENVO-0299 global_var_with_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
