package main

func shape22Mix(a int, b int) int { return a*10 + b }
func appMain(args []string) int {
	if shape22Mix(4, 7) != 47 {
		print("program_shape_22 params\n")
		return 1
	}
	print("PASS\n")
	return 0
}
