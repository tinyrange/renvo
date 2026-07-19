package main

type cell struct {
	value int
	other int
}

func appMain(args []string) int {
	v := cell{value: 0xf0 & 0xaa, other: 1}
	if !(v.value == 0xa0) {
		print("RENVO-0224 bitwise_value_stored_in_struct_field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
