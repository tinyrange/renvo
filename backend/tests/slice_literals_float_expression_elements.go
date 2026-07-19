package main

type renvoSliceFloatScalar = float64

type renvoSliceFloatBounds struct {
	min renvoSliceFloatScalar
	max renvoSliceFloatScalar
}

func appMain(args []string) int {
	bounds := renvoSliceFloatBounds{min: 2, max: 6}
	values := []renvoSliceFloatScalar{bounds.min, (bounds.min + bounds.max) / 2, bounds.max}
	if len(values) != 3 || values[0] != 2 || values[1] != 4 || values[2] != 6 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
