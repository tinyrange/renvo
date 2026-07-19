package main

func renvoMK17Fill(values []int) {
	i := 0
	for i < len(values) {
		values[i] = i + 1
		i += 1
	}
}

func renvoMK17Sum(values []int) int {
	total := 0
	i := 0
	for i < len(values) {
		total += values[i]
		i += 1
	}
	return total
}

func appMain(args []string) int {
	values := make([]int, 4)
	renvoMK17Fill(values)
	if renvoMK17Sum(values) != 10 {
		print("make_slices_helper_fill_and_sum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
