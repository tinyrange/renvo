package main

type functionValueSliceItem struct {
	value int
}

type functionValueSliceCompletion func(source []byte, caret int) []functionValueSliceItem

func functionValueSliceComplete(source []byte, caret int) []functionValueSliceItem {
	return []functionValueSliceItem{{value: len(source) + caret}}
}

func appMain() int {
	var complete functionValueSliceCompletion = functionValueSliceComplete
	items := complete([]byte("abc"), 2)
	if len(items) != 1 || items[0].value != 5 {
		return 1
	}
	print("PASS\n")
	return 0
}
