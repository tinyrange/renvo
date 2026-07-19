package main

type comparisonInner struct {
	Count int
	Tag   string
	Flags [2]bool
}

type comparisonBox struct {
	Value  int
	Name   string
	OK     bool
	Ptr    *int
	Values [2]int
	Inner  comparisonInner
}

type comparisonValues [2]int

type comparisonNamedBox struct {
	Values comparisonValues
}

func comparisonMakeBox(v int) comparisonBox {
	return comparisonBox{v, "a", true, nil, [2]int{1, 2}, comparisonInner{3, "b", [2]bool{true, false}}}
}

func appMain() int {
	x := 1
	left := comparisonBox{1, "a", true, &x, [2]int{1, 2}, comparisonInner{3, "b", [2]bool{true, false}}}
	right := comparisonBox{1, "a", true, &x, [2]int{1, 2}, comparisonInner{3, "b", [2]bool{true, false}}}
	other := comparisonBox{1, "different", true, &x, [2]int{1, 2}, comparisonInner{3, "b", [2]bool{true, false}}}
	namedLeft := comparisonNamedBox{comparisonValues{1, 2}}
	namedRight := comparisonNamedBox{comparisonValues{1, 2}}
	arrayLeft := [2]int{1, 2}
	arrayRight := [2]int{1, 2}
	arrayOther := [2]int{2, 1}
	if left != right {
		print("struct equality failed\n")
		return 1
	}
	if left == other {
		print("struct inequality failed\n")
		return 1
	}
	if arrayLeft != arrayRight || arrayLeft == arrayOther {
		print("array comparison failed\n")
		return 1
	}
	if namedLeft != namedRight {
		print("named composite comparison failed\n")
		return 1
	}
	if comparisonMakeBox(1) != comparisonMakeBox(1) {
		print("returned composite comparison failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
