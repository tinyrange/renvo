package main

func belowNegativeConstant(value int) bool { return value < -1 }
func equalNegativeConstant(value int) bool { return value == -1 }
func aboveNegativeConstant(value int) bool { return value > -1 }

func appMain() int {
	if belowNegativeConstant(-1) || !equalNegativeConstant(-1) || aboveNegativeConstant(-1) {
		panic("negative constant equality")
	}
	if !belowNegativeConstant(-2) || !aboveNegativeConstant(0) || !aboveNegativeConstant(2147483647) {
		panic("negative constant ordering")
	}
	print("PASS\n")
	return 0
}
