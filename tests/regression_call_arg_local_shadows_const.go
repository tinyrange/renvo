package main

const value = 102

func identity(value int) int {
	return pass(value)
}

func pass(value int) int {
	return value
}

func appMain() int {
	if identity(7) == 7 {
		print("PASS\n")
		return 0
	}
	return 1
}
