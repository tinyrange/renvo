package main

type deepOffsetField struct{ Value int }
type middleOffsetField struct{ deepOffsetField }
type shallowOffsetField struct{ Value int }
type outerOffsetField struct {
	middleOffsetField
	shallowOffsetField
}

func appMain(args []string) int {
	var value outerOffsetField
	want := Offsetof(value.shallowOffsetField) + Offsetof(value.shallowOffsetField.Value)
	if Offsetof(value.Value) != want {
		return 1
	}
	value.Value = 42
	if value.shallowOffsetField.Value != 42 || value.Value != 42 || value.middleOffsetField.Value != 0 {
		return 2
	}
	print("PASS\n")
	return 0
}
