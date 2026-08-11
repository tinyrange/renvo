package main

type c89InterfaceStructResult struct {
	first  int
	second int
	third  int
}

type c89InterfaceStructResultSource interface {
	value() c89InterfaceStructResult
}

type c89InterfaceStructResultImpl struct{}

func (c89InterfaceStructResultImpl) value() c89InterfaceStructResult {
	return c89InterfaceStructResult{first: 3, second: 5, third: 7}
}

func appMain() int {
	var source c89InterfaceStructResultSource = c89InterfaceStructResultImpl{}
	value := source.value()
	if value.first != 3 || value.second != 5 || value.third != 7 {
		return 1
	}
	print("PASS\n")
	return 0
}
