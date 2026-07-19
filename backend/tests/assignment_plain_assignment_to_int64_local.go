package main

func appMain(args []string) int {
	var x int64 = 0
	x = 99
	if !(x == 99) {
		print("RENVO-0330 plain_assignment_to_int64_local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
