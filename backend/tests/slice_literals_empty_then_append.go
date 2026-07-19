package main

func appMain(args []string) int {
	values := []int{}
	if len(values) != 0 {
		print("slice_literals_empty_then_append initial length failed\n")
		return 1
	}
	values = append(values, 42)
	if len(values) != 1 || values[0] != 42 {
		print("slice_literals_empty_then_append append failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
