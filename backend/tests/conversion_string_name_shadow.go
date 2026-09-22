package main

func appMain(args []string) int {
	string := func(v int) int { return v + 1 }
	if string(4) != 5 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
