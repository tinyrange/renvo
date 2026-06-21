package main

func appMain(args []string) int {
	dest := make([]byte, 1, 5)
	dest[0] = 'h'
	source := []byte{'i', '!'}
	dest = append(dest, source...)
	if len(dest) != 3 {
		print("append_expansion_into_make_buffer length failed\n")
		return 1
	}
	if dest[0] != 'h' || dest[1] != 'i' || dest[2] != '!' {
		print("append_expansion_into_make_buffer value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
