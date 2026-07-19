package main

func appMain(args []string) int {
	x := 0
	x = 12
	if !(x == 12) {
		print("RENVO-0326 plain_assignment_to_int_local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
