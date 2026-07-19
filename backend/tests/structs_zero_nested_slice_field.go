package main

type zeroNestedSliceFieldInner struct {
	items []int
}

type zeroNestedSliceFieldOuter struct {
	inner zeroNestedSliceFieldInner
	flag  int
}

func appMain(args []string) int {
	var outer zeroNestedSliceFieldOuter
	if len(outer.inner.items) != 0 || outer.flag != 0 {
		print("zero nested slice field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
