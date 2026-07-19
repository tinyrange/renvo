package main

func appMain(args []string) int {
	var values []int
	values = append(values, 64)
	values = append(values, 8)
	values = append(values, 1)
	if values[0]+values[1]+values[2] != 73 {
		print("RENVO-0841 decimal slice index failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
