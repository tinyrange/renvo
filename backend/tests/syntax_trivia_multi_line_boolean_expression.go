package main

func appMain(args []string) int {
	if !((true) &&
		(3 < 4) &&
		(5 != 6)) {
		print("RENVO-0813 multiline bool failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
