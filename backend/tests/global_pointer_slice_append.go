package main

type globalPointerSliceItem struct {
	value int
}

var globalPointerSliceValues []*globalPointerSliceItem

func appendGlobalPointerSliceValue(value int) *globalPointerSliceItem {
	item := &globalPointerSliceItem{value: value}
	globalPointerSliceValues = append(globalPointerSliceValues, item)
	return item
}

func appMain(args []string) int {
	first := appendGlobalPointerSliceValue(1)
	second := appendGlobalPointerSliceValue(2)
	if len(globalPointerSliceValues) != 2 {
		return 1
	}
	if globalPointerSliceValues[0] != first {
		return 1
	}
	if globalPointerSliceValues[1] != second {
		return 1
	}
	print("PASS\n")
	return 0
}
