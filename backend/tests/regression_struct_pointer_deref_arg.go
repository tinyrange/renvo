package main

type structPointerDerefArgBox struct {
	value int
}

func structPointerDerefArgValue(box structPointerDerefArgBox) int {
	return box.value
}

func appMain() int {
	box := structPointerDerefArgBox{value: 42}
	ptr := &box
	copy := *ptr
	if structPointerDerefArgValue(*ptr)+copy.value == 84 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
