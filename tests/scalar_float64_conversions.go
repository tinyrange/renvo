package main

type scalarFloat = float64

func explicitFloatPoint(x int) scalarFloat {
	return scalarFloat(x) + 0.5
}

func appMain(args []string) int {
	positive := explicitFloatPoint(7)
	negative := scalarFloat(-9)
	constant := float64(6)
	if positive != 7.5 || int(positive) != 7 || int(negative) != -9 ||
		constant != 6.0 || int(8.0) != 8 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
