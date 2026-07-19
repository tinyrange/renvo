package main

func appMain(args []string) int {
	var first []int
	first = append(first, 1)
	second := append(first, 2)
	if len(first) != 1 || len(second) != 2 || second[1] != 2 {
		print("RENVO-0575 different append assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
