package main

type assertionSwitchItem struct {
	n int
}

func (value assertionSwitchItem) Number() int {
	return value.n
}

type assertionSwitchPointer struct {
	n int
}

func (value *assertionSwitchPointer) Number() int {
	return value.n
}

type assertionSwitchNumber interface {
	Number() int
}

type assertionSwitchEmbedded interface {
	assertionSwitchNumber
}

type assertionSwitchWrong struct{}

func (assertionSwitchWrong) Number(extra int) int {
	return extra
}

type assertionSwitchWrongEntry struct{}

func (assertionSwitchWrongEntry) Name() string {
	return "wrong"
}

type assertionSwitchWrongFS struct{}

func (assertionSwitchWrongFS) ReadDir(path string) ([]assertionSwitchWrongEntry, bool) {
	return []assertionSwitchWrongEntry{{}}, path == "wrong"
}

type assertionSwitchEntry struct {
	Name string
}

type assertionSwitchFS interface {
	ReadDir(path string) ([]assertionSwitchEntry, bool)
}

type assertionSwitchEmbeddedFS interface {
	assertionSwitchFS
}

type assertionSwitchGoodFS struct{}

func (assertionSwitchGoodFS) ReadDir(path string) ([]assertionSwitchEntry, bool) {
	return []assertionSwitchEntry{{Name: path}}, true
}

func assertionSwitchInterfaceResult() bool {
	var fs assertionSwitchEmbeddedFS = assertionSwitchGoodFS{}
	entries, ok := fs.ReadDir("right")
	return ok && len(entries) == 1 && entries[0].Name == "right"
}

var assertionSwitchCalls int

func assertionSwitchValue(which int) interface{} {
	assertionSwitchCalls++
	if which == 1 {
		return assertionSwitchItem{n: 9}
	}
	if which == 2 {
		return "text"
	}
	if which == 4 {
		return &assertionSwitchPointer{n: 11}
	}
	if which == 5 {
		return assertionSwitchPointer{n: 12}
	}
	if which == 6 {
		return assertionSwitchWrong{}
	}
	return nil
}

func assertionSwitchRecover(value interface{}) (recovered bool) {
	defer func() {
		recovered = recover() != nil
	}()
	_ = value.(string)
	return false
}

func assertionSwitchConcrete(which int) int {
	switch value := assertionSwitchValue(which).(type) {
	case assertionSwitchItem:
		return value.n
	case string:
		return len(value)
	case nil:
		return -1
	default:
		return 0
	}
}

func assertionSwitchInterface(which int) int {
	switch value := assertionSwitchValue(which).(type) {
	case assertionSwitchEmbedded:
		return value.Number()
	case nil:
		return -1
	default:
		return -2
	}
}

func assertionSwitchMultiple(which int) int {
	switch value := assertionSwitchValue(which).(type) {
	case assertionSwitchItem, string:
		item, itemOK := value.(assertionSwitchItem)
		text, textOK := value.(string)
		if itemOK {
			return item.n
		}
		if textOK {
			return len(text)
		}
		return 0
	default:
		return -1
	}
}

func assertionSwitchReturnItem(value interface{}) assertionSwitchItem {
	return value.(assertionSwitchItem)
}

func assertionSwitchReturnComplex(value interface{}) complex128 {
	return value.(complex128)
}

func appMain() int {
	var value interface{} = assertionSwitchItem{n: 7}
	text, ok := value.(string)
	if ok || text != "" {
		print("FAIL comma-ok\n")
		return 1
	}
	item, ok := value.(assertionSwitchItem)
	if !ok || item.n != 7 {
		print("FAIL concrete assertion\n")
		return 1
	}
	number, ok := value.(assertionSwitchNumber)
	if !ok || number.Number() != 7 {
		print("FAIL interface assertion\n")
		return 1
	}
	var pointerValue interface{} = &assertionSwitchPointer{n: 8}
	number, ok = pointerValue.(assertionSwitchNumber)
	if !ok || number.Number() != 8 {
		print("FAIL pointer assertion\n")
		return 1
	}
	var pointerMethodValue interface{} = assertionSwitchPointer{n: 8}
	if _, ok = pointerMethodValue.(assertionSwitchNumber); ok {
		print("FAIL value method set\n")
		return 1
	}
	var wrong interface{} = assertionSwitchWrong{}
	if _, ok = wrong.(assertionSwitchNumber); ok {
		print("FAIL signature match\n")
		return 1
	}
	if !assertionSwitchInterfaceResult() {
		print("FAIL interface result identity\n")
		return 1
	}
	var sequence interface{} = []int{3, 4}
	assertedSequence, ok := sequence.([]int)
	if !ok || len(assertedSequence) != 2 || assertedSequence[1] != 4 {
		print("FAIL slice assertion\n")
		return 1
	}
	if !assertionSwitchRecover(value) {
		print("FAIL mismatch panic\n")
		return 1
	}
	returnedItem := assertionSwitchReturnItem(value)
	complexSource := complex(3, 4)
	var complexValue interface{} = complexSource
	returnedComplex := assertionSwitchReturnComplex(complexValue)
	if returnedItem.n != 7 || real(returnedComplex) != 3 || imag(returnedComplex) != 4 {
		print("FAIL assertion returns\n")
		return 1
	}
	if assertionSwitchConcrete(1) != 9 || assertionSwitchConcrete(2) != 4 || assertionSwitchConcrete(3) != -1 ||
		assertionSwitchInterface(1) != 9 || assertionSwitchInterface(4) != 11 || assertionSwitchInterface(5) != -2 || assertionSwitchInterface(6) != -2 ||
		assertionSwitchMultiple(1) != 9 || assertionSwitchMultiple(2) != 4 || assertionSwitchCalls != 9 {
		print("FAIL type switch\n")
		return 1
	}
	print("PASS\n")
	return 0
}
