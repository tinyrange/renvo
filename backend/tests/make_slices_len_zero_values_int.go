package main

func appMain(args []string) int {
	values := make([]int, 3)
	if len(values) != 3 {
		print("make_slices_len_zero_values_int length failed\n")
		return 1
	}
	if values[0] != 0 || values[1] != 0 || values[2] != 0 {
		print("make_slices_len_zero_values_int zero failed\n")
		return 2
	}
	values[1] = 5
	if values[1] != 5 {
		print("make_slices_len_zero_values_int assign failed\n")
		return 3
	}
	print("PASS\n")
	return 0
}
