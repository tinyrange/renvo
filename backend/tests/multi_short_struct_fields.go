package main

type renvo1046Pair struct {
	left  int
	right int
}

func appMain(args []string) int {
	p := renvo1046Pair{left: 8, right: 3}
	left, right := p.left, p.right
	if left-right != 5 {
		print("RENVO-1046 struct fields short failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
