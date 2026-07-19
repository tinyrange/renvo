package main

func appMain(args []string) int {
	var data []int
	if len(data) == 0 || data[0] == 0 {
		data = append(data, 4)
		data = append(data, 5)
		data = append(data, 6)
	}
	if len(data) != 3 {
		print("RENVO-0845 repeated append length failed\n")
		return 1
	}
	if data[0]+data[1]+data[2] != 15 {
		print("RENVO-0845 repeated append sum failed\n")
		return 2
	}
	print("PASS\n")
	return 0
}
