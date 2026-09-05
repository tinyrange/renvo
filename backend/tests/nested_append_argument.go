package main

func count(s []int) int { return len(s)*10 + s[0] }
func appMain(args []string) int {
	s := []int{1}
	if count(append(s, 2)) != 21 || count(append(s, []int{2, 3}...)) != 31 {
		return 1
	}
	if len(s) != 1 {
		return 2
	}
	print("PASS\n")
	return 0
}
