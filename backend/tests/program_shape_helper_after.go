package main

func appMain(args []string) int {
	if laterShape2(3) != 8 {
		print("program_shape_02 helper\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func laterShape2(x int) int { return x + 5 }
