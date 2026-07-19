package main

type pointerSliceLiteralBox struct {
	value int
}

func pointerSliceLiteralValue(p *pointerSliceLiteralBox) int {
	return p.value
}

func appMain() int {
	a := pointerSliceLiteralBox{value: 40}
	b := pointerSliceLiteralBox{value: 2}
	items := []*pointerSliceLiteralBox{&a, &b}
	if pointerSliceLiteralValue(items[0])+pointerSliceLiteralValue(items[1]) == 42 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
