package main

type pointerSelectorValue struct {
	padding [13]byte
	value   byte
}

func pointerSelectorRead(value **pointerSelectorValue) byte {
	return (*value).value
}

func appMain(args []string) int {
	want := byte(42)
	var inner pointerSelectorValue
	inner.value = want
	outer := &inner
	if pointerSelectorRead(&outer) != want {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
