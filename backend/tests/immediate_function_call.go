package main

func appMain() int {
	value := 41
	func(input int) { value = input + 1 }(value)
	if value != 42 {
		print("FAIL: immediate function call\n")
		return 1
	}
	print("PASS\n")
	return 0
}
