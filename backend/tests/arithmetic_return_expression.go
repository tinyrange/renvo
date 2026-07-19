package main

func checkArithmeticReturn() int {
	return 14 + 6/3 - 5
}
func appMain(args []string) int {
	if !(checkArithmeticReturn() == 11) {
		print("RENVO-0168 arithmetic_return_expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
