package main

const shapeFourBase = 11

func appMain(args []string) int {
	if shapeFourBase+shapeFourTail != 18 {
		print("program_shape_04 globals\n")
		return 1
	}
	print("PASS\n")
	return 0
}

var shapeFourTail int = 7
