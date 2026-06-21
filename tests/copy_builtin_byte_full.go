package main

func appMain(args []string) int {
	source := []byte{'a', 'b', 'c'}
	dest := make([]byte, 3)
	n := copy(dest, source)
	if n != 3 {
		print("copy_builtin_byte_full count failed\n")
		return 1
	}
	if dest[0] != 'a' || dest[2] != 'c' {
		print("copy_builtin_byte_full value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
