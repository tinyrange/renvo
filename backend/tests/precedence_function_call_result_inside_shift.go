package main

func precA(v int) int {
	return v + 1
}
func appMain(args []string) int {
	if !(1<<precA(3) == 16) {
		print("RENVO-0273 function_call_result_inside_shift failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
