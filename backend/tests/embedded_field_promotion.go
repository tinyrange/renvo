package main

type embeddedFieldBase struct {
	value int
}

type embeddedFieldDerived struct {
	embeddedFieldBase
}

func (base *embeddedFieldBase) setValue(value int) {
	base.value = value
}

func appMain() int {
	derived := embeddedFieldDerived{}
	derived.value = 42
	if derived.value != 42 {
		print("FAIL\n")
		return 1
	}
	derived.setValue(84)
	if derived.value != 84 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
