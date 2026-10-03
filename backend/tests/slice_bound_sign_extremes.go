package main

func sliceSignBounds(data []byte, low int, high int, maximum int) (caught bool) {
	defer func() { caught = recover() != nil }()
	result := data[low:high:maximum]
	_ = result
	return
}

func appMain() int {
	data := make([]byte, 4, 8)
	maximum := int(^uint(0) >> 1)
	minimum := -maximum - 1
	if !sliceSignBounds(data, minimum, 0, 0) || !sliceSignBounds(data, 1, minimum, 2) || !sliceSignBounds(data, 0, 0, minimum) {
		panic("negative bounds")
	}
	if !sliceSignBounds(data, maximum, 0, 0) || !sliceSignBounds(data, 0, maximum, 8) || !sliceSignBounds(data, 0, 4, maximum) {
		panic("large bounds")
	}
	if !sliceSignBounds(data, 2, 1, 4) || !sliceSignBounds(data, 1, 4, 3) || !sliceSignBounds(data, 0, 4, 9) {
		panic("reversed or capacity bounds")
	}
	if sliceSignBounds(data, 0, 0, 0) || sliceSignBounds(data, 1, 6, 7) || sliceSignBounds(data, 8, 8, 8) {
		panic("valid bounds")
	}
	print("PASS\n")
	return 0
}
