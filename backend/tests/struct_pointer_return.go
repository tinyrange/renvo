package main

type pointerReturnValue struct {
	left  int
	right int
}

func pointerReturn(value *pointerReturnValue) pointerReturnValue {
	return *value
}

func appMain(args []string) int {
	value := pointerReturnValue{left: 3, right: 5}
	result := pointerReturn(&value)
	if result.left != 3 || result.right != 5 {
		print("FAIL: struct pointer return was corrupted\n")
		return 1
	}
	print("PASS\n")
	return 0
}
