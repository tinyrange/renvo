package main

func appMain(args []string) int {
	var values = [2]int{1, 2}
	empty := [0]int{}
	if len(values) != 2 || values[0] != 1 || values[1] != 2 || len(empty) != 0 {
		print("arrays_inferred_var_and_zero_length failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
