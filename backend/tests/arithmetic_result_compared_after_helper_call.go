package main

func scale(v int) int {
	return v * 3
}
func appMain(args []string) int {
	if !(scale(7+5) == 36) {
		print("RENVO-0175 arithmetic_result_compared_after_helper_call failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
