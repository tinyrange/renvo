package main

func runtimeStringNegativeIndexRecovered() (recovered bool) {
	defer func() {
		recovered = recover() != nil
	}()
	text := "x"
	index := -1
	_ = text[index]
	return false
}

func appMain() int {
	if !runtimeStringNegativeIndexRecovered() {
		return 1
	}
	print("PASS\n")
	return 0
}
