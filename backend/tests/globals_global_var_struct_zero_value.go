package main

type renvo0689Pair struct {
	left  int
	right int
}

var renvo0689Value renvo0689Pair

func appMain(args []string) int {
	if renvo0689Value.left != 0 {
		print("RENVO-0689 struct zero left failed\n")
		return 1
	}
	if renvo0689Value.right != 0 {
		print("RENVO-0689 struct zero right failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
