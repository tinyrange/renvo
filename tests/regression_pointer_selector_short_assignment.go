package main

type pointerSelectorValue struct {
	value int
}

type pointerSelectorHolder struct {
	value *pointerSelectorValue
}

func appMain() int {
	value := pointerSelectorValue{value: 42}
	holder := pointerSelectorHolder{value: &value}
	got := holder.value
	if got.value != 42 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
