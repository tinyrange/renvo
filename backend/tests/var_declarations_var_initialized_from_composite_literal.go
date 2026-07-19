package main

type cell struct {
	value int
	other int
}

func appMain(args []string) int {
	var v cell = cell{value: 7, other: 8}
	if !(v.value+v.other == 15) {
		print("RENVO-0296 var_initialized_from_composite_literal failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
