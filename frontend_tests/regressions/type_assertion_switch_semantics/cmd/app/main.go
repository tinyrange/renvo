package main

import "example.com/renvotests/regressions/type_assertion_switch_semantics/lib"

func recoverMismatch(value interface{}) (recovered bool) {
	defer func() {
		recovered = recover() != nil
	}()
	_ = value.(string)
	return false
}

func concreteSwitch(which int) int {
	switch value := lib.Choose(which).(type) {
	case lib.Item:
		return value.Value
	case string:
		return len(value)
	case nil:
		return -1
	default:
		return 0
	}
}

func interfaceSwitch(which int) int {
	switch value := lib.Choose(which).(type) {
	case lib.EmbeddedNumber:
		return value.Number()
	case nil:
		return -1
	default:
		return -2
	}
}

func multipleSwitch(which int) int {
	switch value := lib.Choose(which).(type) {
	case lib.Item, string:
		item, itemOK := value.(lib.Item)
		text, textOK := value.(string)
		if itemOK {
			return item.Value
		}
		if textOK {
			return len(text)
		}
	}
	return -1
}

func returnItem(value interface{}) lib.Item {
	return value.(lib.Item)
}

func returnComplex(value interface{}) complex128 {
	return value.(complex128)
}

func main() {
	var value interface{} = lib.Item{Value: 7}
	text, ok := value.(string)
	item, itemOK := value.(lib.Item)
	number, numberOK := value.(lib.Number)
	var pointer interface{} = &lib.Pointer{Value: 8}
	pointerNumber, pointerOK := pointer.(lib.Number)
	var pointerValue interface{} = lib.Pointer{Value: 8}
	_, pointerValueOK := pointerValue.(lib.Number)
	var wrong interface{} = lib.Wrong{}
	_, wrongOK := wrong.(lib.Number)
	var sequence interface{} = []int{3, 4}
	assertedSequence, sequenceOK := sequence.([]int)
	returnedItem := returnItem(value)
	complexSource := complex(3, 4)
	var complexValue interface{} = complexSource
	returnedComplex := returnComplex(complexValue)
	if ok || text != "" || !itemOK || item.Value != 7 || !numberOK || number.Number() != 7 ||
		!pointerOK || pointerNumber.Number() != 8 || pointerValueOK || wrongOK ||
		!sequenceOK || len(assertedSequence) != 2 || assertedSequence[1] != 4 || returnedItem.Value != 7 ||
		real(returnedComplex) != 3 || imag(returnedComplex) != 4 || !recoverMismatch(value) ||
		concreteSwitch(1) != 9 || concreteSwitch(2) != 4 || concreteSwitch(3) != -1 ||
		interfaceSwitch(1) != 9 || interfaceSwitch(4) != 11 || interfaceSwitch(5) != -2 || interfaceSwitch(6) != -2 ||
		multipleSwitch(1) != 9 || multipleSwitch(2) != 4 || lib.Calls != 9 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
}
