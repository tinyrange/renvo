package main

type typedFloatConstantScalar = float64

const typedFloatConstantRowHeight typedFloatConstantScalar = 34

func appMain(args []string) int {
	y := typedFloatConstantScalar(5)
	y += typedFloatConstantRowHeight
	if y != 39 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
