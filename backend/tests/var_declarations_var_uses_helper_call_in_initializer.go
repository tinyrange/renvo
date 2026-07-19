package main

func helperVarDecl() int {
	return 21
}
func appMain(args []string) int {
	var x int = helperVarDecl()
	if !(x == 21) {
		print("RENVO-0294 var_uses_helper_call_in_initializer failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
