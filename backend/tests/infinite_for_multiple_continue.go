package main

func appMain(args []string) int {
	i := 0
	sum := 0
	for {
		i = i + 1
		if i == 1 {
			continue
		}
		if i == 3 {
			continue
		}
		if i > 4 {
			break
		}
		sum = sum + i
	}
	if sum != 6 {
		print("RENVO-0446 multiple continue failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
