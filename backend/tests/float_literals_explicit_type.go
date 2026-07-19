package main

func appMain(args []string) int {
	var x float64 = 7.25
	if x != 7.25 {
		print("float_literals_15 explicit\n")
		return 1
	}
	print("PASS\n")
	return 0
}
