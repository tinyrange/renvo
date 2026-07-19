package main

type renvo1018Pair struct {
	left  int
	right int
}

func renvo1018Fields(p renvo1018Pair) (int, int) {
	return p.left, p.right
}

func appMain(args []string) int {
	left, right := renvo1018Fields(renvo1018Pair{left: 3, right: 11})
	if left*right != 33 {
		print("RENVO-1018 struct field pair failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
