package main

func appMain(args []string) int {
	values := []float64{}
	values = append(values, 9)
	if len(values) != 1 {
		print("FAIL\n")
		return 1
	}
	if int(values[0]) != 9 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
