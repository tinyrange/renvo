package main

type exportedStructFieldBox struct {
	Value int
}

func appMain() int {
	box := exportedStructFieldBox{Value: 42}
	if box.Value == 42 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
