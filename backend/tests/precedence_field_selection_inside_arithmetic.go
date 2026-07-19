package main

type cell struct {
	value int
	other int
}

func appMain(args []string) int {
	v := cell{value: 8, other: 3}
	if !(v.value+v.other*2 == 14) {
		print("RENVO-0269 field_selection_inside_arithmetic failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
