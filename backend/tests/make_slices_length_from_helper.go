package main

func renvoMK12Len() int {
	return 5
}

func appMain(args []string) int {
	values := make([]int, renvoMK12Len())
	i := 0
	for i < len(values) {
		values[i] = i * 2
		i += 1
	}
	if len(values) != 5 || values[4] != 8 {
		print("make_slices_length_from_helper failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
