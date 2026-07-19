package main

const constTopA = 6
const constTopB = 5

func appMain(args []string) int {
	if !(constTopA*constTopB == 30) {
		print("RENVO-0169 arithmetic_with_top_level_constants failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
