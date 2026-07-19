package main

func appMain(args []string) int {
	total := 0
	for _, value := range [2]int{3, 4} {
		total += value
	}
	count := 0
	for range []int{8, 9, 10} {
		count++
	}
	if total != 7 || count != 3 {
		print("range_array_literal_and_slice_no_variables failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
