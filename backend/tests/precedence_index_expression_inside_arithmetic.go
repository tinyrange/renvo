package main

func appMain(args []string) int {
	s := []int{}
	s = append(s, 4)
	s = append(s, 9)
	if !(s[0]+s[1]*2 == 22) {
		print("RENVO-0270 index_expression_inside_arithmetic failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
