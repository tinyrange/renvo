package main

type namedFloatScalar float64

type namedFloatPoint struct {
	y namedFloatScalar
}

func namedFloatCoordinate(position namedFloatPoint, scale namedFloatScalar, row int) namedFloatScalar {
	return position.y + namedFloatScalar(row)*scale
}

func appMain(args []string) int {
	for row := 0; row < 7; row++ {
		got := int(namedFloatCoordinate(namedFloatPoint{y: 2}, 1, row))
		if got != row+2 {
			print("FAIL\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
