package main

func mutateArrayParameter(values [2]int) int {
	values[0] = 9
	return values[0] + values[1]
}

func appMain(args []string) int {
	values := [2]int{1, 2}
	if mutateArrayParameter(values) != 11 || values[0] != 1 {
		print("array_parameter_value_copy failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
