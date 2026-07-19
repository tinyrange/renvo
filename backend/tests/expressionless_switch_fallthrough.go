package main

func appMain() int {
	value := 0
	switch {
	case value != 0:
		value = 9
	case value == 0:
		value = 1
		fallthrough
	case value == 99:
		value += 2
	default:
		value = 10
	}
	if value == 3 {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
