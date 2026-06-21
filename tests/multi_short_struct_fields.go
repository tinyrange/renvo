package main

type rtg1046Pair struct {
	left  int
	right int
}

func appMain(args []string) int {
	p := rtg1046Pair{left: 8, right: 3}
	left, right := p.left, p.right
	if left-right != 5 {
		print("RTG-1046 struct fields short failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
