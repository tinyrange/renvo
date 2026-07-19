package main

func appMain(args []string) int {
	i := 0
	sum := 0
	for {
		if i < 3 {
			sum = sum + i
		} else {
			break
		}
		i = i + 1
	}
	if sum != 3 {
		print("RENVO-0434 if else infinite failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
