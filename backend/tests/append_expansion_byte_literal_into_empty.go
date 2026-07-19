package main

func appMain(args []string) int {
	source := []byte{'o', 'k'}
	var dest []byte
	dest = append(dest, source...)
	if len(dest) != 2 {
		print("append_expansion_byte_literal_into_empty length failed\n")
		return 1
	}
	if dest[0] != 'o' || dest[1] != 'k' {
		print("append_expansion_byte_literal_into_empty value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
