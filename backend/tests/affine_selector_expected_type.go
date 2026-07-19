package main

type affineSelectorMatrix struct {
	xx float64
	yy float64
	w  float64
}

type affineSelectorSurface struct {
	transform affineSelectorMatrix
}

func setAffineSelectorTransform(surface *affineSelectorSurface, transform *affineSelectorMatrix) {
	surface.transform = *transform
	surface.transform.w = 1
}

func setAffineSelectorLinear(surface *affineSelectorSurface, xx, yy float64) {
	surface.transform.xx = xx
	surface.transform.yy = yy
	surface.transform.w = 1
}

func appMain(args []string) int {
	surface := &affineSelectorSurface{}
	identity := affineSelectorMatrix{xx: 1, yy: 1, w: 1}
	setAffineSelectorTransform(surface, &identity)
	if int(surface.transform.xx) != 1 || int(surface.transform.yy) != 1 || int(surface.transform.w) != 1 {
		print("FAIL\n")
		return 1
	}
	setAffineSelectorLinear(surface, 1, 1)
	if int(surface.transform.xx) != 1 || int(surface.transform.yy) != 1 || int(surface.transform.w) != 1 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
