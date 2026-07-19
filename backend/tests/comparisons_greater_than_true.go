package main

func appMain(args []string) int {
	if !(9 > 2) {
		print("RENVO-0183 greater_than_true failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
