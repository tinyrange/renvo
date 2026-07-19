package main

type functionFieldItem struct{ value int }
type functionFieldCompletion func(source []byte, caret int) []functionFieldItem

type functionFieldEditor struct {
	complete functionFieldCompletion
}

type functionFieldProvider struct{}

func (p *functionFieldProvider) complete(source []byte, caret int) []functionFieldItem {
	return []functionFieldItem{{value: len(source) + caret}}
}

func appMain() int {
	var editor functionFieldEditor
	var provider functionFieldProvider
	editor.complete = provider.complete
	items := editor.complete([]byte("abc"), 2)
	if len(items) != 1 || items[0].value != 5 {
		return 1
	}
	print("PASS\n")
	return 0
}
