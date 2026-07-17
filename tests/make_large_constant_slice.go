package main

func appMain(args []string) int {
	const count = 65536
	values := make([]int, count)
	if len(values) != count || cap(values) != count {
		print("FAIL\n")
		return 1
	}
	for i := 0; i < len(values); i += 4095 {
		if values[i] != 0 {
			print("FAIL\n")
			return 1
		}
		values[i] = i + 1
	}
	if values[0] != 1 || values[65520] != 65521 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
