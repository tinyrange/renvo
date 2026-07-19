package main

func appMain(args []string) int {
	if !(true || false && false) {
		print("RENVO-0257 and_before_or failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
