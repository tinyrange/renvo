package main

func appMain(args []string) int {
	s := make([]int, 0)
	if s == nil {
		return 1
	}
	s = append(s, 1)
	if s == nil {
		return 2
	}
	var t []int = make([]int, 0)
	if t == nil {
		return 3
	}
	var n []int
	if n != nil {
		return 4
	}
	print("PASS\n")
	return 0
}
