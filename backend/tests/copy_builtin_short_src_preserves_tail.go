package main

func appMain(args []string) int {
	dest := []byte{'x', 'y', 'z'}
	source := []byte{'a'}
	n := copy(dest, source)
	if n != 1 {
		print("copy_builtin_short_src_preserves_tail count failed\n")
		return 1
	}
	if dest[0] != 'a' || dest[1] != 'y' || dest[2] != 'z' {
		print("copy_builtin_short_src_preserves_tail value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
