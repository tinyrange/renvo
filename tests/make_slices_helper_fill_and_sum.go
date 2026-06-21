package main

func rtgMK17Fill(values []int) {
	i := 0
	for i < len(values) {
		values[i] = i + 1
		i += 1
	}
}

func rtgMK17Sum(values []int) int {
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
	rtgMK17Fill(values)
	if rtgMK17Sum(values) != 10 {
		print("make_slices_helper_fill_and_sum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
