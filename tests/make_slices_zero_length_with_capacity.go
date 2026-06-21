package main

func appMain(args []string) int {
	buf := make([]byte, 0, 4)
	if len(buf) != 0 {
		print("make_slices_zero_length_with_capacity initial length failed\n")
		return 1
	}
	buf = append(buf, 'a')
	buf = append(buf, 'b')
	if len(buf) != 2 || buf[0] != 'a' || buf[1] != 'b' {
		print("make_slices_zero_length_with_capacity append failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
