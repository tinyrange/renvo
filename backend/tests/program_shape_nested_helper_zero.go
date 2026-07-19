package main

func shape12inner(x int) int { return x + 4 }
func shape12outer(x int) int { return shape12inner(x) * 2 }
func appMain(args []string) int {
	if shape12outer(3) != 14 {
		print("program_shape_12 nested\n")
		return 1
	}
	print("PASS\n")
	return 0
}
