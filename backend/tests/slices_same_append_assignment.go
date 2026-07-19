package main

func appMain(args []string) int {
	var xs []int
	// Same variable receives append result.
	xs = append(xs,
		17)
	if len(xs) != 1 || xs[0] != 17 {
		print("RENVO-0574 same append assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
