package main

func appMain(args []string) int {
	names := []string{"red", "blue"}
	pick := 1
	if len(names) != 2 {
		print("slice_literals_string_values_select length failed\n")
		return 1
	}
	if names[0] != "red" || names[pick] != "blue" {
		print("slice_literals_string_values_select value failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
