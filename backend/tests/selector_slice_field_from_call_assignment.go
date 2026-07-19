package main

type renvoSelectorSliceCallBox struct {
	items []int
}

func renvoSelectorSliceCallMake() []int {
	var out []int
	out = append(out, 6)
	return out
}

func appMain(args []string) int {
	var box renvoSelectorSliceCallBox
	box.items = renvoSelectorSliceCallMake()
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
