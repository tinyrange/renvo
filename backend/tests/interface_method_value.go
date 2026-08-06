package main

type interfaceMethodValue interface {
	apply(int) int
}

type interfaceMethodValueAdder int

func (value interfaceMethodValueAdder) apply(input int) int {
	return int(value) + input
}

func appMain() int {
	var operation interfaceMethodValue = interfaceMethodValueAdder(7)
	apply := operation.apply
	if apply(5) != 12 {
		print("FAIL: interface method value\n")
		return 1
	}
	print("PASS\n")
	return 0
}
