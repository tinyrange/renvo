package main

type rtgSelectorSliceCallBox struct {
	items []int
}

func rtgSelectorSliceCallMake() []int {
	var out []int
	out = append(out, 6)
	return out
}

func appMain(args []string) int {
	var box rtgSelectorSliceCallBox
	box.items = rtgSelectorSliceCallMake()
	if len(box.items) != 1 {
		print("selector slice field call length failed\n")
		return 1
	}
	if box.items[0] != 6 {
		print("selector slice field call value failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
