package main

func appMain(args []string) int {
	if shape6a(2) != 17 {
		print("program_shape_06 chain\n")
		return 1
	}
	print("PASS\n")
	return 0
}
func shape6a(x int) int { return shape6b(x) + 3 }
func shape6b(x int) int { return shape6c(x) * 2 }
func shape6c(x int) int { return x + 5 }
