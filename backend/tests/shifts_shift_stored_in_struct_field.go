package main

type cell struct {
	value int
	other int
}

func appMain(args []string) int {
	v := cell{value: 1 << 5, other: 2}
	if !(v.value == 32) {
		print("RENVO-0245 shift_stored_in_struct_field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
