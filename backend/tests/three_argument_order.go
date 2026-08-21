package main

func encodeThreeArguments(first int, second int, third int) int {
	return first*100 + second*10 + third
}

func appMain(args []string) int {
	if encodeThreeArguments(1, 2, 3) == 123 {
		print("PASS\n")
		return 0
	}
	return 1
}
