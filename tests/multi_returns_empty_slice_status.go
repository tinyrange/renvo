package main

func rtg1016Empty() ([]int, bool) {
	var xs []int
	return xs, false
}

func appMain(args []string) int {
	xs, ok := rtg1016Empty()
	if ok || len(xs) != 0 {
		print("RTG-1016 empty slice status failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
