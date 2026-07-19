package main

func appMain(args []string) int {
	values := []int{2, 4, 8, 16}
	sum := 0
	i := 0
	for i < len(values) {
		sum += values[i]
		i += 1
	}
	if sum != 30 {
		print("slice_literals_int_values_loop_sum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
