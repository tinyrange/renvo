package main

func appMain(args []string) int {
	dest := make([]int, 3)
	source := []int{7, 8}
	if copy(dest, source) != 2 {
		print("copy_builtin_return_used_in_condition count failed\n")
		return 1
	}
	if dest[0]+dest[1]+dest[2] != 15 {
		print("copy_builtin_return_used_in_condition value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
