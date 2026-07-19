package main

type implicitFloatPoint struct {
	x float64
}

var implicitGlobalFloat float64 = 13
var implicitGlobalPoint = implicitFloatPoint{x: 17}

func implicitFloatIdentity(x float64) float64 { return x }

func implicitFloatReturn() float64 { return 11 }

func appMain(args []string) int {
	point := implicitFloatPoint{x: 40}
	var assigned float64
	assigned = 19
	mixed := 20.0 + 3
	if int(implicitFloatIdentity(40)) != 40 || int(point.x) != 40 ||
		int(assigned) != 19 || int(implicitFloatReturn()) != 11 ||
		int(implicitGlobalFloat) != 13 || int(implicitGlobalPoint.x) != 17 ||
		int(mixed) != 23 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
