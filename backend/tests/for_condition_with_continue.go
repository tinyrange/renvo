package main

func appMain(args []string) int {
	i := 0
	sum := 0
	for i < 5 {
		i = i + 1
		if i == 3 {
			continue
		}
		sum = sum + i
	}
	if sum != 12 {
		print("RENVO-0387 condition continue failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
