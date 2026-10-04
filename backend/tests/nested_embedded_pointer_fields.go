package main

type nestedPointerLeaf struct {
	Padding int
	Value   int
}

type nestedPointerMiddle struct {
	Padding int
	*nestedPointerLeaf
}

type nestedPointerOuter struct {
	Padding int
	*nestedPointerMiddle
}

func appMain() int {
	leaf := nestedPointerLeaf{17, 42}
	middle := nestedPointerMiddle{23, &leaf}
	outer := nestedPointerOuter{29, &middle}
	if outer.Value != 42 {
		return 1
	}
	outer.Value = 51
	if leaf.Value != 51 {
		return 2
	}
	pointer := &outer
	if pointer.Value != 51 {
		return 3
	}
	print("PASS\n")
	return 0
}
