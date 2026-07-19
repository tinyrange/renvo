package main

func appMain(args []string) int {
	i := 1
	sum := 0
	for {
		sum = sum + i*2
		i = i + 1
		if i == 5 {
			break
		}
	}
	if sum != 20 {
		print("RENVO-0448 checksum infinite failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
