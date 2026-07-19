package main

func appMain(args []string) int {
	n := 0
	for {
		n = n + 1
		if n == 4 {
			break
		}
	}
	if n != 4 {
		print("RENVO-0428 increment break failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
