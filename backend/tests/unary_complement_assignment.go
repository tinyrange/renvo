package main

func appMain(args []string) int {
	value := 7
	value = ^value
	if value != -8 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
