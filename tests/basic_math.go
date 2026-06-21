package main

func appMain(args []string) int {
	// test addition
	if 1+2 != 3 {
		print("addition failed\n")
		return 1
	}

	// test subtraction
	if 5-3 != 2 {
		print("subtraction failed\n")
		return 1
	}

	// test multiplication
	if 4*6 != 24 {
		print("multiplication failed\n")
		return 1
	}

	// test division
	if 10/2 != 5 {
		print("division failed\n")
		return 1
	}

	print("PASS\n")
	return 0
}
