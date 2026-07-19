package main

const firstPart = 17
const secondPart = 25

func appMain(args []string) int {
	a := firstPart
	b := secondPart
	total := a + b
	if total != 42 {
		print("RENVO-0846 temporary values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
