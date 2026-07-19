package main

func appMain(args []string) int {
	dest := []byte{'a'}
	source := []byte{}
	dest = append(dest, source...)
	if len(dest) != 1 {
		print("append_expansion_empty_source_no_change length failed\n")
		return 1
	}
	if dest[0] != 'a' {
		print("append_expansion_empty_source_no_change value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
