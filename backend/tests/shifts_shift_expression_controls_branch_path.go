package main

func appMain(args []string) int {
	x := 1 << 4
	if x == 16 {
		// expected
	} else {
		print("RENVO-0250 shift_expression_controls_branch_path failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
