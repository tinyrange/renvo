package main

type descriptorLeaf struct {
	text string
	data []int
}

type descriptorOuter struct {
	padding int
	leaf    descriptorLeaf
}

func descriptorCheck(prefix int, text string, values []int, suffix int) bool {
	return prefix == 11 && suffix == 29 && text == "descriptor" &&
		len(values) == 2 && cap(values) == 4 && values[0] == 7 && values[1] == 13
}

func descriptorEmpty(text string, values []int) bool {
	return len(text) == 0 && len(values) == 0 && cap(values) == 0 && values == nil
}

func appMain() int {
	values := make([]int, 2, 4)
	values[0], values[1] = 7, 13
	outer := descriptorOuter{padding: 101, leaf: descriptorLeaf{text: "descriptor", data: values}}
	pointer := &outer
	if !descriptorCheck(11, outer.leaf.text, pointer.leaf.data, 29) ||
		!descriptorCheck(11, pointer.leaf.text, outer.leaf.data, 29) {
		print("FAIL\n")
		return 1
	}
	zero := descriptorLeaf{}
	if !descriptorEmpty(zero.text, zero.data) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
