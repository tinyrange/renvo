package main

func appMain(args []string) int {
	ok := false
	ok = true
	if !(ok) {
		print("RENVO-0327 plain_assignment_to_bool_local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
