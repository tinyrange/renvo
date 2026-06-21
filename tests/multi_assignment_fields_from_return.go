package main

type rtg1028Pair struct {
	left  int
	right int
}

func rtg1028Values() (int, int) {
	return 10, 6
}

func appMain(args []string) int {
	p := rtg1028Pair{}
	p.left, p.right = rtg1028Values()
	if p.left-p.right != 4 {
		print("RTG-1028 fields from return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
