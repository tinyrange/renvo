package main

const arrayOpaqueStructBound = int(^uint(0) >> 2)

type arrayOpaqueStructBlock struct {
	Items [arrayOpaqueStructBound]struct{}
}

func appMain() int {
	var value *arrayOpaqueStructBlock
	if len(value.Items) != arrayOpaqueStructBound {
		return 1
	}
	print("PASS\n")
	return 0
}
