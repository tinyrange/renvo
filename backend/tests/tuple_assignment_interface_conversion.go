package main

var tupleAssignmentGlobal any

func tupleAssignmentMixed() (int, string, int) { return 42, "value", 43 }

func appMain() int {
	var first int
	var text any
	var last any
	first, text, last = tupleAssignmentMixed()
	if first != 42 || text.(string) != "value" || last.(int) != 43 {
		panic("local tuple conversion")
	}
	var record struct{ Text any }
	items := make([]any, 1)
	var pointerValue any
	pointer := &pointerValue
	first, record.Text, items[0] = tupleAssignmentMixed()
	if first != 42 || record.Text.(string) != "value" || items[0].(int) != 43 {
		panic("address tuple conversion")
	}
	first, *pointer, tupleAssignmentGlobal = tupleAssignmentMixed()
	if first != 42 || pointerValue.(string) != "value" || tupleAssignmentGlobal.(int) != 43 {
		panic("pointer/global tuple conversion")
	}
	text, newLast := text, 44
	if text.(string) != "value" || newLast != 44 {
		panic("existing interface descriptor")
	}
	print("PASS\n")
	return 0
}
