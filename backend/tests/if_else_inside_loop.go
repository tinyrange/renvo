package main

func appMain(args []string) int {
	sum := 0
	for i := 0; i < 4; i = i + 1 {
		if i > 1 {
			sum = sum + i
		}
	}
	if sum != 5 {
		print("RENVO-0372 if loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
