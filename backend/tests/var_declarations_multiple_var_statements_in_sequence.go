package main

func appMain(args []string) int {
	var a int = 1
	var b int = 2
	var c int = 3
	if !(a+b+c == 6) {
		print("RENVO-0297 multiple_var_statements_in_sequence failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
