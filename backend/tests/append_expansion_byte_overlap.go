package main

func appMain(args []string) int {
	values := make([]byte, 5)
	values[0] = 1
	values[1] = 2
	values[2] = 3
	values[3] = 4
	result := append(values[:2], values[1:4]...)
	if len(result) != 5 || result[0] != 1 || result[1] != 2 || result[2] != 2 || result[3] != 3 || result[4] != 4 {
		return 1
	}
	print("PASS\n")
	return 0
}
