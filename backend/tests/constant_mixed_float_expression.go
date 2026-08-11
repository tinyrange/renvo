package main

const mixedFloatConstant = 10 + 0.0

func appMain() int {
	if mixedFloatConstant != 10.0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
