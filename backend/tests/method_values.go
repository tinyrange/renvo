package main

type methodValueNumber struct {
	value int
}

func (number methodValueNumber) add(delta int) int {
	return number.value + delta
}

func appMain() int {
	number := methodValueNumber{value: 4}
	bound := number.add
	if bound(3) != 7 {
		return 1
	}
	expression := methodValueNumber.add
	if expression(methodValueNumber{value: 5}, 6) != 11 {
		return 1
	}
	print("PASS\n")
	return 0
}
