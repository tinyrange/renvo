package main

func appMain(args []string) int {
	n := 0
	for n == 0 {
		n = 1
	}
	if n != 1 {
		print("RENVO-0380 one iteration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
