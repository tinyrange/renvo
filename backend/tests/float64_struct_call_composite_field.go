package main

type float64StructCallRect struct {
	minX float64
	minY float64
	maxX float64
	maxY float64
}

type float64StructCallNode struct {
	bounds float64StructCallRect
}

func float64StructCallMakeRect(x, y, width, height float64) float64StructCallRect {
	return float64StructCallRect{minX: x, minY: y, maxX: x + width, maxY: y + height}
}

func appMain(args []string) int {
	base := float64StructCallMakeRect(1, 2, 3, 4)
	index := 2
	node := float64StructCallNode{bounds: float64StructCallMakeRect(base.minX+1, base.minY+float64(index*2), base.maxX-base.minX, base.maxY-base.minY)}
	if node.bounds.maxX != 5 || node.bounds.maxY != 10 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
