package main

func shapeTen() int { return 10 }

func appMain(args []string) int {

	if shapeTen() != 10 {
		print("program_shape_10 blanks\n")
		return 1
	}

	print("PASS\n")
	return 0
}
