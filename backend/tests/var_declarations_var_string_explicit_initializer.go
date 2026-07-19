package main

func appMain(args []string) int {
	var x string = "ok"
	if !(x == "ok") {
		print("RENVO-0282 var_string_explicit_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
