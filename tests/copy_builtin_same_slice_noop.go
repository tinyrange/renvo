package main

func appMain(args []string) int {
	buf := []byte{'n', 'o'}
	n := copy(buf, buf)
	if n != 2 {
		print("copy_builtin_same_slice_noop count failed\n")
		return 1
	}
	if buf[0] != 'n' || buf[1] != 'o' {
		print("copy_builtin_same_slice_noop value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
