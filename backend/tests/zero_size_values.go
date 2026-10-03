package main

type ZeroSizeEmpty struct{}
type ZeroSizeRecord struct {
	Before int
	Empty  ZeroSizeEmpty
	After  int
}

var zeroSizeCalls int
var zeroSizeGlobal ZeroSizeEmpty
var zeroSizeNeighbor int = 91

func zeroSizeValue() ZeroSizeEmpty {
	zeroSizeCalls++
	return ZeroSizeEmpty{}
}

func zeroSizeTuple() (ZeroSizeEmpty, int) {
	return zeroSizeValue(), 42
}

func zeroSizeArgument(value ZeroSizeEmpty, after int) int { return after }

func zeroSizeVariadic(values ...ZeroSizeEmpty) int { return len(values) }

func appMain(args []string) int {
	var empty ZeroSizeEmpty
	var array [10]ZeroSizeEmpty
	var none [0]int
	if Sizeof(empty) != 0 || Sizeof(array) != 0 || Sizeof(none) != 0 {
		panic("zero-size query")
	}
	if Alignof(empty) != 1 || Alignof(array) != 1 || Alignof(none) != Alignof(int(0)) {
		panic("zero-size alignment")
	}
	var record ZeroSizeRecord
	record.Before = 17
	record.After = 51
	if Offsetof(record.Empty) != Offsetof(record.After) {
		panic("empty field consumes storage")
	}
	record.Empty = zeroSizeValue()
	array[9] = zeroSizeValue()
	if uintptr(&array[0]) != uintptr(&array[9]) {
		panic("zero-size array stride")
	}
	var next int
	record.Empty, next = zeroSizeTuple()
	if next != 42 || record.Before != 17 || record.After != 51 {
		panic("empty assignment overwrites neighbor")
	}
	if zeroSizeArgument(zeroSizeValue(), 73) != 73 || zeroSizeCalls != 4 {
		panic("zero-size call argument or evaluation")
	}
	zeroSizeGlobal = zeroSizeValue()
	if zeroSizeNeighbor != 91 {
		panic("empty global assignment overwrites neighbor")
	}
	values := make([]ZeroSizeEmpty, 3, 5)
	if values == nil || len(values) != 3 || cap(values) != 5 || uintptr(&values[0]) != uintptr(&values[2]) {
		panic("zero-size slice allocation or stride")
	}
	values = append(values, zeroSizeValue())
	literal := []ZeroSizeEmpty{zeroSizeValue(), zeroSizeValue()}
	values = append(values, literal...)
	if len(values) != 6 || cap(values) < 6 || copy(values, literal) != 2 {
		panic("zero-size append or copy")
	}
	part := values[2:4:5]
	if len(part) != 2 || cap(part) != 3 || uintptr(&part[0]) != uintptr(&values[0]) {
		panic("zero-size subslice stride")
	}
	if zeroSizeVariadic(zeroSizeValue(), zeroSizeValue()) != 2 || zeroSizeCalls != 10 {
		panic("zero-size variadic arguments")
	}
	allocated := new(ZeroSizeEmpty)
	if allocated == nil {
		panic("nil zero-size allocation")
	}
	literalPointer := &ZeroSizeEmpty{}
	if literalPointer == nil {
		panic("nil zero-size literal pointer")
	}
	print("PASS\n")
	return 0
}
