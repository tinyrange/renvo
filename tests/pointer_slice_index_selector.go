package main

type pointerSliceSelectorItem struct {
	first int
	value int
}

func appMain(args []string) int {
	first := &pointerSliceSelectorItem{first: 1, value: 2}
	second := &pointerSliceSelectorItem{first: 3, value: 4}
	values := []*pointerSliceSelectorItem{first, second}
	index := 1

	if values[0].value != 2 {
		return 1
	}
	if values[index].first != 3 {
		return 1
	}
	if values[index].value != 4 {
		return 1
	}
	print("PASS\n")
	return 0
}
