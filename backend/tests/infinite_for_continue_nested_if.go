package main

func appMain(args []string) int {
	i := 0
	sum := 0
	for {
		i = i + 1
		if i < 3 {
			if true {
				continue
			}
		}
		sum = sum + i
		if i == 4 {
			break
		}
	}
	if sum != 7 {
		print("RENVO-0444 nested continue failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
