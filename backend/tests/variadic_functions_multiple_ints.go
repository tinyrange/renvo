package main

func renvoVF44Sum(values ...int) int {
	total := 0
	i := 0
	for i < len(values) {
		total += values[i]
		i += 1
	}
	return total
}

func appMain(args []string) int {
	if renvoVF44Sum(1, 2, 3, 4) != 10 {
		print("variadic_functions_multiple_ints failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
