package main

type rtgWideFieldRect struct {
	minX float64
	minY float64
	maxX float64
	maxY float64
}

func rtgWideFieldMakeRect() rtgWideFieldRect {
	return rtgWideFieldRect{maxX: 45, maxY: 80}
}

func rtgWideFieldIntersect(a, b rtgWideFieldRect) rtgWideFieldRect {
	if b.minX > a.minX {
		a.minX = b.minX
	}
	if b.minY > a.minY {
		a.minY = b.minY
	}
	if b.maxX < a.maxX {
		a.maxX = b.maxX
	}
	if b.maxY < a.maxY {
		a.maxY = b.maxY
	}
	return a
}

func appMain(args []string) int {
	r := rtgWideFieldMakeRect()
	r = rtgWideFieldIntersect(r, rtgWideFieldMakeRect())
	if r.minX != 0 || r.minY != 0 || r.maxX != 45 || r.maxY != 80 {
		print("FAIL: 32-bit struct argument corrupted wide fields\n")
		return 1
	}
	print("PASS\n")
	return 0
}
