package main

func appMain(args []string) int {
	ok := true
	n := 0
	for ok {
		n = n + 1
		if n == 3 {
			ok = false
		}
	}
	if n != 3 {
		print("RENVO-0378 bool loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
