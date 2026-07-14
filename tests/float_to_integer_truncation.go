package main

func appMain(args []string) int {
	negativeQuarter := -0.25
	negativeHalf := -1.5
	negativeThreeQuarters := -2.75
	positiveHalf := 3.5

	if int(negativeQuarter) != 0 {
		return 1
	}
	if int(negativeHalf) != -1 {
		return 1
	}
	if int32(negativeThreeQuarters) != -2 {
		return 1
	}
	if int16(positiveHalf) != 3 {
		return 1
	}
	print("PASS\n")
	return 0
}
