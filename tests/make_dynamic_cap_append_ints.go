package main

func appMain(args []string) int {
	count := 5000
	values := make([]int, 0, count)
	i := 0
	for i < count {
		values = append(values, i)
		i++
	}
	if len(values) != count {
		print("FAIL\n")
		return 1
	}
	if values[0] != 0 {
		print("FAIL\n")
		return 1
	}
	if values[1234] != 1234 {
		print("FAIL\n")
		return 1
	}
	if values[4999] != 4999 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
