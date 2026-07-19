package main

func appMain(args []string) int {
	n := 4
	for {
		n = n - 1
		if n == 0 {
			break
		}
	}
	if n != 0 {
		print("RENVO-0429 decrement break failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
