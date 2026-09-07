package main

func shadowResult() (x int) {
	defer func() {}()
	if true {
		x := 11
		return x
	}
	return
}

func appMain(args []string) int {
	if shadowResult() != 11 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
