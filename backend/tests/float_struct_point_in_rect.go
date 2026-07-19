package main

type hitScalar float64

type hitRect struct {
	MinX hitScalar
	MinY hitScalar
	MaxX hitScalar
	MaxY hitScalar
}

func hitPointInRect(x, y hitScalar, rect hitRect) bool {
	return x >= rect.MinX && x < rect.MaxX && y >= rect.MinY && y < rect.MaxY
}

func appMain() int {
	rect := hitRect{MinX: 0, MinY: 0, MaxX: 260, MaxY: 700}
	if !hitPointInRect(145, 270, rect) || hitPointInRect(300, 270, rect) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
