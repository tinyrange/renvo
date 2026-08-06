package main

func appMain() int {
	values := make(map[string]int)
	length := -1
	values["new"], length = 7, len(values)
	if length != 0 || len(values) != 1 || values["new"] != 7 {
		print("FAIL: map entry committed before RHS\n")
		return 1
	}
	print("PASS\n")
	return 0
}
