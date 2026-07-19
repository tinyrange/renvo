package main

func appMain() int {
	value := complex(3, 4)
	if real(value) != 3 || imag(value) != 4 {
		return 1
	}
	value = value + (2 + 1i)
	if real(value) != 5 || imag(value) != 5 {
		return 1
	}
	print("PASS\n")
	return 0
}
