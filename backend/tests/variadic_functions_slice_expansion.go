package main

func renvoVF46Sum(values ...int) int {
	total := 0
	i := 0
	for i < len(values) {
		total += values[i]
		i += 1
	}
	return total
}

func appMain(args []string) int {
	values := []int{3, 5, 7}
	if renvoVF46Sum(values...) != 15 {
		print("variadic_functions_slice_expansion failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
