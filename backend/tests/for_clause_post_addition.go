package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 5; i += 2 {
		sum = sum + i
	}
	if sum != 6 {
		print("RENVO-0405 post addition failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
