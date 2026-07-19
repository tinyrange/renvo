package main

func appMain(args []string) int {
	if !renvo0530Less(3, 9) {
		print("RENVO-0530 bool return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func renvo0530Less(a int, b int) bool {
	return a < b
}
