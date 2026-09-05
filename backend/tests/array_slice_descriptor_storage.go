package main

func size(s []int) int { return len(s) }
func appMain(args []string) int {
	var n []int
	a := [1]int{4}
	if size(a[:0]) != 0 {
		return 1
	}
	if n != nil {
		return 2
	}
	print("PASS\n")
	return 0
}
