package main

func appMain(args []string) int {
	i := 0
	for i < 3 {
		i = i + 1
	}
	if !(i == 3) {
		print("RENVO-0190 comparison_used_in_for_condition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
