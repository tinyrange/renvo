package main

func appMain(args []string) int {
	i := 0
	sum := 0
	for i < 5 {
		if i > 1 {
			sum = sum + i
		}
		i = i + 1
	}
	if sum != 9 {
		print("RENVO-0388 nested if loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
