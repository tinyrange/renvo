package main

func shape24Tab() int { return 24 }

func appMain(args []string) int {
	if shape24Tab() != 24 {
		print("program_shape_24 tabs\n")
		return 1
	}
	print("PASS\n")
	return 0
}
