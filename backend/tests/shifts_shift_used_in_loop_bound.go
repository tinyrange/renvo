package main

func appMain(args []string) int {
	limit := 1 << 3
	i := 0
	for i < limit {
		i = i + 2
	}
	if !(i == 8) {
		print("RENVO-0243 shift_used_in_loop_bound failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
