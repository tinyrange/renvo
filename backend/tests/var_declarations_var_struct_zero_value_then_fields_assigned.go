package main

type cell struct {
	value int
	other int
}

func appMain(args []string) int {
	var v cell
	v.value = 5
	v.other = 6
	if !(v.value+v.other == 11) {
		print("RENVO-0285 var_struct_zero_value_then_fields_assigned failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
