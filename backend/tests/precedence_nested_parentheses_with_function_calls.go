package main

func precA(v int) int {
	return v + 1
}
func appMain(args []string) int {
	if !((precA(3)+precA(4))*2 == 18) {
		print("RENVO-0267 nested_parentheses_with_function_calls failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
