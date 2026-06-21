package main

func rtgAE32Tail() []byte {
	return []byte{'T', 'G'}
}

func appMain(args []string) int {
	dest := []byte{'R'}
	dest = append(dest, rtgAE32Tail()...)
	if len(dest) != 3 {
		print("append_expansion_from_function_return length failed\n")
		return 1
	}
	if dest[0] != 'R' || dest[1] != 'T' || dest[2] != 'G' {
		print("append_expansion_from_function_return value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
