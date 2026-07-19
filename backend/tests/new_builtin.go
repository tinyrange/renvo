package main

type newBuiltinBox struct {
	value int
	text  string
}

func appMain() int {
	first := new(int)
	second := new(int)
	if first == nil || second == nil || first == second || *first != 0 || *second != 0 {
		print("new scalar initialization failed\n")
		return 1
	}
	*first = 41
	*second = *first + 1
	if *first != 41 || *second != 42 {
		print("new scalar storage failed\n")
		return 1
	}
	box := new(newBuiltinBox)
	if box == nil || box.value != 0 || box.text != "" {
		print("new struct initialization failed\n")
		return 1
	}
	box.value = *second
	box.text = "ready"
	if box.value != 42 || box.text != "ready" {
		print("new struct storage failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
