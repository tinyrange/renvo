package main

func integerComplexComponents() int {
	return real(complex(17, 1)) + imag(complex(2, 25))
}

func appMain() int {
	if integerComplexComponents() != 42 {
		return 1
	}
	print("PASS\n")
	return 0
}
