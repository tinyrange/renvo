package main

func helperStringByte(value string, index int) int {
	return int(value[index])
}

func appMain(args []string) int {
	index := 0
	if len(args) > 1 {
		index = 2
		if args[1] == "negative" {
			index = -1
		}
	}
	if helperStringByte("x", index) != 120 {
		return 1
	}
	print("PASS\n")
	return 0
}
