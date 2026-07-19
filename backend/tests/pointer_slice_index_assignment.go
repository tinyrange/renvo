package main

type pointerSliceItem struct {
	value int
}

func appMain(args []string) int {
	first := &pointerSliceItem{value: 1}
	second := &pointerSliceItem{value: 2}
	values := []*pointerSliceItem{first, second}
	index := 0
	values[index] = values[1]

	if values[0] != second {
		return 1
	}
	print("PASS\n")
	return 0
}
