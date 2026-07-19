package main

func appMain(args []string) int {
	one := float64(1)
	if one != 1 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
