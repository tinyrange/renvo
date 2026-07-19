package main

type shape16Pair struct {
	left  int
	right int
}

var shape16Global shape16Pair = shape16Pair{left: 12, right: 5}

func shape16Sum() int { return shape16Global.left + shape16Global.right }
func appMain(args []string) int {
	if shape16Sum() != 17 {
		print("program_shape_16 struct\n")
		return 1
	}
	print("PASS\n")
	return 0
}
