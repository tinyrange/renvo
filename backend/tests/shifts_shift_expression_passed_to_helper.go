package main

func acceptShift(v int) int {
	return v
}
func appMain(args []string) int {
	if !(acceptShift(3<<2) == 12) {
		print("RENVO-0249 shift_expression_passed_to_helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
