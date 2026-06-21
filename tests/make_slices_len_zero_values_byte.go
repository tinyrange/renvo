package main

func appMain(args []string) int {
	buf := make([]byte, 4)
	if len(buf) != 4 {
		print("make_slices_len_zero_values_byte length failed\n")
		return 1
	}
	if buf[0] != 0 || buf[3] != 0 {
		print("make_slices_len_zero_values_byte zero failed\n")
		return 2
	}
	buf[2] = 'q'
	if buf[2] != 'q' {
		print("make_slices_len_zero_values_byte assign failed\n")
		return 3
	}
	print("PASS\n")
	return 0
}
