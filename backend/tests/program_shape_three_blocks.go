package main

func appMain(args []string) int {
	if shape18a(1) != 10 {
		print("program_shape_18 chain\n")
		return 1
	}
	print("PASS\n")
	return 0
}

func shape18a(x int) int { return shape18b(x + 2) }

func shape18b(x int) int { return shape18c(x) + 4 }

func shape18c(x int) int { return x + 3 }
