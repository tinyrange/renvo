package main

const arith02C = 17

func appMain(args []string) int {
	a := 25
	if a+arith02C != 42 {
		print("arithmetic_02 addconst\n")
		return 1
	}
	print("PASS\n")
	return 0
}
