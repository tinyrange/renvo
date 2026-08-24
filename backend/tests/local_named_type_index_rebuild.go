package main

func buildLocalNamedType() int {
	type localValue struct {
		value int
	}
	item := localValue{value: 7}
	return item.value
}

func localValue() int {
	return 9
}

func callAfterLocalNamedType() int {
	return localValue()
}

func appMain(args []string) int {
	if buildLocalNamedType() != 7 || callAfterLocalNamedType() != 9 {
		return 1
	}
	print("PASS\n")
	return 0
}
