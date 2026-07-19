package main

func appMain(args []string) int {
	value := "P\x41\x53\x53\x0a"
	if len(value) != 5 || value[0] != 'P' || value[1] != 'A' || value[2] != 'S' || value[3] != 'S' || value[4] != '\n' {
		print("FAIL\n")
		return 1
	}
	print(value)
	return 0
}
