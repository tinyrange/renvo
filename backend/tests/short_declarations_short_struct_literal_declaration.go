package main

type cell struct {
	value int
	other int
}

func appMain(args []string) int {
	v := cell{value: 1, other: 2}
	if !(v.value+v.other == 3) {
		print("RENVO-0307 short_struct_literal_declaration failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
