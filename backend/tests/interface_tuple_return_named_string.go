package main

type interfaceTupleReturnValue interface {
	Type() string
}

type interfaceTupleReturnString string

func (interfaceTupleReturnString) Type() string { return "string" }

func interfaceTupleReturnAdd(a, b interfaceTupleReturnValue) (interfaceTupleReturnValue, error) {
	as := a.(interfaceTupleReturnString)
	bs := b.(interfaceTupleReturnString)
	return as + bs, nil
}

func interfaceTupleReturnEqual(a, b interfaceTupleReturnValue) bool {
	switch x := a.(type) {
	case interfaceTupleReturnString:
		y, ok := b.(interfaceTupleReturnString)
		return ok && x == y
	}
	return false
}

func appMain() int {
	value, err := interfaceTupleReturnAdd(interfaceTupleReturnString("PA"), interfaceTupleReturnString("SS\n"))
	if err == nil && value.Type() == "string" && interfaceTupleReturnEqual(value, interfaceTupleReturnString("PASS\n")) {
		print("PASS\n")
		return 0
	}
	print("FAIL\n")
	return 1
}
