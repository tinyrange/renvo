package main

func appMain(args []string) int {
	source := []int{1, 2, 3, 4}
	dest := make([]int, 2)
	n := copy(dest, source)
	if n != 2 {
		print("copy_builtin_int_short_dst count failed\n")
		return 1
	}
	if dest[0]+dest[1] != 3 {
		print("copy_builtin_int_short_dst value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
