package main

func appMain(args []string) int {
	a := 1
	b := 2
	c := 3
	a, b, c = c, a, b
	if a*100+b*10+c != 312 {
		print("RENVO-1022 three local assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
