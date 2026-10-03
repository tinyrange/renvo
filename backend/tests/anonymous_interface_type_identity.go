package main

type readerAlias = interface{ Read() int }
type namedReader interface{ Read() int }
type intAlias = int
type firstReader interface{ Read(value intAlias) int }
type secondReader interface{ Read(other int) int }
type combinedAlias = interface {
	firstReader
	secondReader
	Write(values ...string) error
}

func sameAnonymousInterface(value any) bool {
	_, ok := value.(*interface{ Read() int })
	return ok
}

func appMain(args []string) int {
	var value *readerAlias
	if !sameAnonymousInterface(value) {
		return 1
	}
	var array [2]*readerAlias
	var arrayValue any = array
	if _, ok := arrayValue.([2]*interface{ Read() int }); !ok {
		return 2
	}
	var slice []*readerAlias
	var sliceValue any = slice
	if _, ok := sliceValue.([]*interface{ Read() int }); !ok {
		return 3
	}
	var record struct{ Child readerAlias }
	var recordValue any = record
	if _, ok := recordValue.(struct{ Child interface{ Read() int } }); !ok {
		return 4
	}
	var fn func(readerAlias) *readerAlias
	var functionValue any = fn
	if _, ok := functionValue.(func(interface{ Read() int }) *interface{ Read() int }); !ok {
		return 5
	}
	var combined *combinedAlias
	var combinedValue any = combined
	if _, ok := combinedValue.(*interface {
		Write(...string) error
		Read(int) int
	}); !ok {
		return 6
	}
	if _, ok := combinedValue.(*interface {
		Read(int) int
		Write([]string) error
	}); ok {
		return 7
	}
	if _, ok := combinedValue.(*interface {
		Write(...string) interface{ Error() string }
		Read(int) int
	}); ok {
		return 8
	}
	if _, ok := combinedValue.(*interface {
		Write(...string) error
		Read(string) int
	}); ok {
		return 9
	}
	var defined *namedReader
	if sameAnonymousInterface(defined) {
		return 10
	}
	var errorPointer *error
	var errorValue any = errorPointer
	if _, ok := errorValue.(*interface{ Error() string }); ok {
		return 11
	}
	if _, ok := errorValue.(*error); !ok {
		return 12
	}
	print("PASS\n")
	return 0
}
