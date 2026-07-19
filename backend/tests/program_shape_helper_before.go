package main

func earlierShape3(x int) int { return x * 4 }

func appMain(args []string) int {
	if earlierShape3(6) != 24 {
		print("program_shape_03 helper\n")
		return 1
	}
	print("PASS\n")
	return 0
}
