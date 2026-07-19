package main

func appMain(args []string) int {
	if !(3 < 7) {
		print("RENVO-0179 less_than_true failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
