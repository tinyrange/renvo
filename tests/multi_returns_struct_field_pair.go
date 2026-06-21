package main

type rtg1018Pair struct {
	left  int
	right int
}

func rtg1018Fields(p rtg1018Pair) (int, int) {
	return p.left, p.right
}

func appMain(args []string) int {
	left, right := rtg1018Fields(rtg1018Pair{left: 3, right: 11})
	if left*right != 33 {
		print("RTG-1018 struct field pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
