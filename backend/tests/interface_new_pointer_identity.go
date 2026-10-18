package main

var boxedNewPointer interface{}

func assignNewPointer() { p := new(int); *p = 731; boxedNewPointer = p }
func appMain() int {
	assignNewPointer()
	p, ok := boxedNewPointer.(*int)
	if !ok || *p != 731 {
		return 1
	}
	print("PASS\n")
	return 0
}
