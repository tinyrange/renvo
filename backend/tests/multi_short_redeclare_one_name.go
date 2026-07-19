package main

func appMain(args []string) int {
	a := 5
	a, b := a+1, 7
	if a != 6 || b != 7 {
		print("RENVO-1041 short redeclare one failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
