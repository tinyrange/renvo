package main

type groupedLocalPair struct {
	left  int
	right int
}

func appMain(args []string) int {
	var first, second int
	second = 7
	if first != 0 || second != 7 {
		return 1
	}

	var left, right string = "left", "right"
	if left != "left" || right != "right" {
		return 1
	}

	var one, two groupedLocalPair
	two.right = 9
	if one.left != 0 || one.right != 0 || two.left != 0 || two.right != 9 {
		return 1
	}

	print("PASS\n")
	return 0
}
