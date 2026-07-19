package main

func appMain(args []string) int {
	i := 0
	sum := 0
	for i < 4 {
		i, sum = i+1, sum+i
	}
	if i != 4 || sum != 6 {
		print("RENVO-1035 loop step assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
