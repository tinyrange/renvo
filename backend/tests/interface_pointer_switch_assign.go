package main

func interfacePointerSwitchAssign(destination any, value any) {
	switch out := destination.(type) {
	case *any:
		*out = value
	}
}

func appMain() int {
	var got any
	interfacePointerSwitchAssign(&got, 7)
	value, ok := got.(int)
	if ok && value == 7 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
