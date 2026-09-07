package main

func appMain(args []string) int {
	int := func(s string) int { return len(s) }
	if int("abcd") != 4 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
