package main

type renvo1023Pair struct {
	left  int
	right int
}

func appMain(args []string) int {
	p := renvo1023Pair{left: 4, right: 9}
	p.left, p.right = p.right, p.left
	if p.left != 9 || p.right != 4 {
		print("RENVO-1023 struct field swap failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
