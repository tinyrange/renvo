package main

func renvoVF43Sum(values ...int) int {
	total := 0
	i := 0
	for i < len(values) {
		total += values[i]
		i += 1
	}
	return total
}

func appMain(args []string) int {
	if renvoVF43Sum() != 0 {
		print("variadic_functions_zero_args failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
