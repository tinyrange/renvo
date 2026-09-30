package main

func pushPopArguments(a int, b int, c int, d int, e int, f int, g int, h int) int {
	return a + b*2 + c*3 + d*4 + e*5 + f*6 + g*7 + h*8
}

func pushPopChoose(value int) int {
	if value < 0 {
		return -value
	}
	return value + 1
}

func appMain() int {
	values := []int{3, -5, 11, -7}
	for i := 0; i < len(values); i++ {
		value := values[i]
		chosen := pushPopChoose(value)
		got := pushPopArguments(value, 17, chosen, -3, value+2, 0, 257, 9)
		want := value + 34 + chosen*3 - 12 + (value+2)*5 + 257*7 + 72
		if got != want {
			print("argument transfer failed\n")
			return 1
		}
		if pushPopArguments(1, 2, 3, 4, 5, 6, 7, 8) != 204 {
			print("constant argument transfer failed\n")
			return 1
		}
		if (value+chosen)*(chosen-value) != chosen*chosen-value*value {
			print("nested stack expression failed\n")
			return 1
		}
	}
	print("PASS\n")
	return 0
}
