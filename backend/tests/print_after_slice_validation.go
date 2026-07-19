package main

func appMain(args []string) int {
	var s []int
	s = append(s, 4)
	s = append(s, 7)
	if len(s) != 2 {
		print("RENVO-0713 slice diagnostic length failed\n")
		return 1
	}
	if s[0]+s[1] != 11 {
		print("RENVO-0713 slice diagnostic value failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
