package main

func appMain(args []string) int {
	ok := 11 >= 10
	if !(ok) {
		print("RENVO-0188 comparison_result_assigned_to_bool failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
