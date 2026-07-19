package main

func renvoSL9Next(x int) int {
	return x + 3
}

func appMain(args []string) int {
	base := 4
	values := []int{base, base * 2, renvoSL9Next(base)}
	if len(values) != 3 {
		print("slice_literals_expression_elements length failed\n")
		return 1
	}
	if values[0]+values[1]+values[2] != 19 {
		print("slice_literals_expression_elements value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
