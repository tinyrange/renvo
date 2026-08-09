package main

func appMain() int {
	value := 2
	pointer := &value
	(*pointer)--
	if value != 1 {
		return 1
	}
	print("PASS\n")
	return 0
}
