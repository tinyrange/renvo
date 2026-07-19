package main

func floatLit17(x float64) bool { return x == 9.75 }
func appMain(args []string) int {
	if !floatLit17(9.75) {
		print("float_literals_17 param\n")
		return 1
	}
	print("PASS\n")
	return 0
}
