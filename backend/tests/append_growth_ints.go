package main

func appMain(args []string) int {
	var values []int
	i := 0
	for i < 300 {
		values = append(values, i)
		i++
	}
	if len(values) != 300 {
		print("FAIL\n")
		return 1
	}
	if values[0] != 0 {
		print("FAIL\n")
		return 1
	}
	if values[127] != 127 {
		print("FAIL\n")
		return 1
	}
	if values[299] != 299 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
