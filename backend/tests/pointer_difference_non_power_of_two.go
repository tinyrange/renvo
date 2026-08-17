package main

type pointerDifferenceTriple struct {
	a byte
	b byte
	c byte
}

func __c_pointer_diff_non_power_of_two(left *pointerDifferenceTriple, right *pointerDifferenceTriple) int64 {
	return 0
}

func appMain(args []string) int {
	var values [2]pointerDifferenceTriple
	if __c_pointer_diff_non_power_of_two(&values[1], &values[0]) != 1 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
