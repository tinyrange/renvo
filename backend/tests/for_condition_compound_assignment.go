package main

func appMain(args []string) int {
	i := 0
	sum := 0
	for i < 4 {
		sum += 2
		i += 1
	}
	if sum != 8 {
		print("RENVO-0390 compound loop failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
