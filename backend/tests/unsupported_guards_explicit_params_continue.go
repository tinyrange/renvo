package main

func addExplicit(a int, b int) int { return a + b }

func appMain(args []string) int {
	sum := 0
	i := 0
	for i < 5 {
		i = i + 1
		if i == 2 {
			continue
		}
		sum += addExplicit(i, 1)
	}
	if sum != 17 {
		print("RENVO-0836 explicit params continue failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
