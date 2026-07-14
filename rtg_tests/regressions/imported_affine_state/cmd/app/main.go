package main

import "example.com/rtgtests/regressions/imported_affine_state/affine"

func main() {
	surface := affine.NewSurface(8, 8)
	surface.SetTranslation(4, 4)
	identity := affine.Identity()
	surface.SetTransform(&identity)
	surface.FillMarker()
	if surface.Pixels[3] != 255 {
		print("FAIL\n")
		return
	}
	surface.Pixels[3] = 0
	surface.ResetTransform()
	surface.SetLinear(1, 0, 0, 1)
	surface.FillMarker()
	if surface.Pixels[3] == 255 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
