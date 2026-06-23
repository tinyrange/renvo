package main

type rtgNestedSelectorLeaf struct {
	value int
}

type rtgNestedSelectorMiddle struct {
	ptr *rtgNestedSelectorLeaf
}

type rtgNestedSelectorRoot struct {
	mid rtgNestedSelectorMiddle
}

func rtgNestedSelectorValue(root *rtgNestedSelectorRoot) int {
	return root.mid.ptr.value
}

func appMain(args []string, env []string) int {
	var leaf rtgNestedSelectorLeaf
	var root rtgNestedSelectorRoot
	leaf.value = 42
	root.mid.ptr = &leaf
	if rtgNestedSelectorValue(&root) == 42 {
		print("PASS\n")
	}
	return 0
}
