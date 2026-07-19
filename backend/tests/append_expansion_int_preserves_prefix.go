package main

func appMain(args []string) int {
	dest := []int{1}
	source := []int{2, 3}
	dest = append(dest, source...)
	if len(dest) != 3 {
		print("append_expansion_int_preserves_prefix length failed\n")
		return 1
	}
	if dest[0]+dest[1]+dest[2] != 6 {
		print("append_expansion_int_preserves_prefix value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
