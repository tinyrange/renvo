package main

type renvo1028Pair struct {
	left  int
	right int
}

func renvo1028Values() (int, int) {
	return 10, 6
}

func appMain(args []string) int {
	p := renvo1028Pair{}
	p.left, p.right = renvo1028Values()
	if p.left-p.right != 4 {
		print("RENVO-1028 fields from return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
