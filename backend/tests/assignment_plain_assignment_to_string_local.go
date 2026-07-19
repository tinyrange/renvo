package main

func appMain(args []string) int {
	s := "bad"
	s = "good"
	if !(s == "good") {
		print("RENVO-0328 plain_assignment_to_string_local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
