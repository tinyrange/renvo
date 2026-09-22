package main

const value = 7

func appMain() int {
	var value = value + 1
	if value != 8 {
		return 1
	}
	print("PASS\n")
	return 0
}
