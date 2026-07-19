package main

func appMain(args []string) int {
	a := 0
	b := 1
	for a < 4 {
		a = a + 1
		b = b + a
	}
	if b != 11 {
		print("RENVO-0395 two local loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
