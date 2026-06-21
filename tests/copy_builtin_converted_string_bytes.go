package main

func appMain(args []string) int {
	source := []byte("copy")
	dest := make([]byte, 4)
	n := copy(dest, source)
	if n != 4 {
		print("copy_builtin_converted_string_bytes count failed\n")
		return 1
	}
	if dest[0] != 'c' || dest[3] != 'y' {
		print("copy_builtin_converted_string_bytes value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
