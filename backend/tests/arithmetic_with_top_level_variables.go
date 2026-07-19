package main

var arithGlobalA int = 18
var arithGlobalB int = 7

func appMain(args []string) int {
	if !(arithGlobalA-arithGlobalB == 11) {
		print("RENVO-0170 arithmetic_with_top_level_variables failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
