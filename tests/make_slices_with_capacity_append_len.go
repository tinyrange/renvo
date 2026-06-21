package main

func appMain(args []string) int {
	values := make([]int, 2, 5)
	values[0] = 3
	values[1] = 4
	values = append(values, 5)
	if len(values) != 3 {
		print("make_slices_with_capacity_append_len length failed\n")
		return 1
	}
	if values[0]+values[1]+values[2] != 12 {
		print("make_slices_with_capacity_append_len value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
