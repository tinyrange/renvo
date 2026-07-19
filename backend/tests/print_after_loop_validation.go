package main

func appMain(args []string) int {
	sum := 0
	i := 0
	for i < 5 {
		sum += i
		i = i + 1
	}
	if sum != 10 {
		print("RENVO-0711 loop diagnostic failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
