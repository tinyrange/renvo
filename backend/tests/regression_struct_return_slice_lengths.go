package main

type lengthBox struct {
	left  []int
	right []int
}

func buildLengthBox() lengthBox {
	left := make([]int, 0, 3)
	left = append(left, 11)
	right := make([]int, 0, 5)
	right = append(right, 21)
	right = append(right, 31)
	var box lengthBox
	box.left = left
	box.right = right
	return box
}

func appMain(args []string) int {
	box := buildLengthBox()
	if len(box.left) != 1 {
		print("RENVO-1123 returned struct first slice length failed\n")
		return 1
	}
	if len(box.right) != 2 {
		print("RENVO-1123 returned struct second slice length failed\n")
		return 1
	}
	if box.left[0] != 11 || box.right[0] != 21 || box.right[1] != 31 {
		print("RENVO-1123 returned struct slice values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
