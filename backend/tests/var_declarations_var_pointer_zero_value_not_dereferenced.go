package main

func appMain(args []string) int {
	var x int = 4
	var p *int
	if x == 4 {
		p = &x
	}
	if !(*p == 4) {
		print("RENVO-0286 var_pointer_zero_value_not_dereferenced failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
