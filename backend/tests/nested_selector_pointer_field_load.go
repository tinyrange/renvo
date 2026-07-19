package main

type renvoNestedSelectorLeaf struct {
	value int
}

type renvoNestedSelectorMiddle struct {
	ptr *renvoNestedSelectorLeaf
}

type renvoNestedSelectorRoot struct {
	mid renvoNestedSelectorMiddle
}

func renvoNestedSelectorValue(root *renvoNestedSelectorRoot) int {
	return root.mid.ptr.value
}

func appMain(args []string, env []string) int {
	var leaf renvoNestedSelectorLeaf
	var root renvoNestedSelectorRoot
	leaf.value = 42
	root.mid.ptr = &leaf
	if renvoNestedSelectorValue(&root) == 42 {
		print("PASS\n")
	}
	return 0
}
