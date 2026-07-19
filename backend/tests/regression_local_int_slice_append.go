package main

func appMain(args []string) int {
	values := make([]int, 0, 2)
	values = append(values, 11)
	values = append(values, 31)
	if len(values) != 2 {
		print("RENVO-1122 local int slice append length failed\n")
		return 1
	}
	if values[0] != 11 || values[1] != 31 {
		print("RENVO-1122 local int slice append values failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
