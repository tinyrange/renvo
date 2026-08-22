package main

func appMain(args []string) int {
	value := 1
	if value != 0 {
		value = value + 1
	}
	{
		value = value + 2
	}
	if value != 4 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
