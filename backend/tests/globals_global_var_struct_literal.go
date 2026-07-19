package main

type renvo0690Pair struct {
	left  int
	right int
}

var renvo0690Value renvo0690Pair = renvo0690Pair{left: 4, right: 8}

func appMain(args []string) int {
	if renvo0690Value.left+renvo0690Value.right != 12 {
		print("RENVO-0690 struct literal global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
