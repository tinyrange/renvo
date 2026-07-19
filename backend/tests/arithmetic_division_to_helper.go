package main

func arith13(x int) bool { return x == 8 }
func appMain(args []string) int {
	if !arith13(43 / 5) {
		print("arithmetic_13 helper\n")
		return 1
	}
	print("PASS\n")
	return 0
}
