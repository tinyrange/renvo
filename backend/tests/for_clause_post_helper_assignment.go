package main

func renvo0408Next(x int) int { return x + 3 }
func appMain(args []string) int {
	sum := 0
	for i := 0; i < 7; i = renvo0408Next(i) {
		sum = sum + i
	}
	if sum != 9 {
		print("RENVO-0408 post helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
