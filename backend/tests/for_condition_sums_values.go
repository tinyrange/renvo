package main

func appMain(args []string) int {
	i := 1
	sum := 0
	for i <= 4 {
		sum = sum + i
		i = i + 1
	}
	if sum != 10 {
		print("RENVO-0381 sum loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
