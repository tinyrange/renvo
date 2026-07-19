package main

func renvo0479Mix(a int, b int) int { return a*10 + b }
func appMain(args []string) int {
	if renvo0479Mix(3, 4) != 34 {
		print("RENVO-0479 two param failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
