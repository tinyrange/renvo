package main

type cell struct {
	value int
	other int
}

func appMain(args []string) int {
	v := cell{value: 6, other: 2}
	x := v.value
	if !(x == 6) {
		print("RENVO-0323 short_declaration_from_field_selection failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
