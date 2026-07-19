package main

func appMain(args []string) int {
	n := 0
	for n < 0 {
		n = n + 1
	}
	if n != 0 {
		print("RENVO-0379 immediate exit failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
