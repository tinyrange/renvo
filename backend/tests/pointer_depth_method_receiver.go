package main

type pointerDepthMethodValue struct {
	padding [8]byte
	value   byte
}

func (value *pointerDepthMethodValue) __c_ptr_8_value() *byte {
	return &value.value
}

func pointerDepthMethodRead(value **pointerDepthMethodValue) byte {
	return *value.__c_ptr_8_value()
}

func appMain(args []string) int {
	want := byte(42)
	var inner pointerDepthMethodValue
	inner.value = want
	outer := &inner
	if pointerDepthMethodRead(&outer) != want {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
