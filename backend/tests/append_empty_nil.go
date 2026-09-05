package main

func nilSlice(s []int) bool { return s == nil }
func appMain(args []string) int {
	var s []int
	if !nilSlice(append(s, []int{}...)) { return 1 }
	print("PASS\n")
	return 0
}
