package main

type pointerCompositeWindow struct {
	Width  int
	Height int
}

func newPointerCompositeWindow(width int, height int) *pointerCompositeWindow {
	w := &pointerCompositeWindow{Width: width, Height: height}
	return w
}

func disturbPointerCompositeStack(seed int) int {
	a := pointerCompositeWindow{Width: seed + 10, Height: seed + 20}
	b := pointerCompositeWindow{Width: seed + 30, Height: seed + 40}
	return a.Width + a.Height + b.Width + b.Height
}

func appMain() int {
	first := newPointerCompositeWindow(123, 456)
	second := newPointerCompositeWindow(789, 1011)
	if disturbPointerCompositeStack(7) == 0 {
		return 1
	}
	if first == nil || second == nil || first == second {
		return 2
	}
	if first.Width != 123 || first.Height != 456 {
		return 3
	}
	if second.Width != 789 || second.Height != 1011 {
		return 4
	}
	print("PASS\n")
	return 0
}
