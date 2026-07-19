package main

func renvoVF47Add(base int, values ...int) int {
	total := base
	i := 0
	for i < len(values) {
		total += values[i]
		i += 1
	}
	return total
}

func appMain(args []string) int {
	if renvoVF47Add(10, 1, 2, 3) != 16 {
		print("variadic_functions_fixed_prefix failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
