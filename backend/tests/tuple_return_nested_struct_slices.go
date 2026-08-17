package main

type tupleReturnInner struct {
	name     string
	goName   string
	typeID   int
	offset   int
	align    int
	bitWidth int
	emit     bool
}

type tupleReturnOuter struct {
	expr     []byte
	receiver []byte
	field    tupleReturnInner
	typeID   int
	end      int
	bitfield bool
}

func tupleReturnBuild(expr []byte, receiver []byte, field tupleReturnInner) (tupleReturnOuter, bool) {
	return tupleReturnOuter{expr: expr, receiver: receiver, field: field, typeID: field.typeID,
		end: len(expr), bitfield: true}, true
}

func appMain(args []string) int {
	field := tupleReturnInner{name: "field", goName: "Field", typeID: 17, offset: 8, align: 4, bitWidth: 3, emit: true}
	value, ok := tupleReturnBuild([]byte{1, 2}, []byte{3, 4, 5}, field)
	if !ok || len(value.expr) != 2 || value.expr[1] != 2 || len(value.receiver) != 3 || value.receiver[2] != 5 ||
		value.field.name != "field" || value.field.goName != "Field" || value.typeID != 17 || value.end != 2 || !value.bitfield {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
