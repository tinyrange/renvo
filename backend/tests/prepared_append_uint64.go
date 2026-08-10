package main

func appMain(args []string) int {
	values := make([]uint64, 0, 2)
	values = append(values, 0x123456789abcdef0)
	if len(values) != 1 || cap(values) != 2 || values[0] != 0x123456789abcdef0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
