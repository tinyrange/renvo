package main

func appMain(args []string) int {
	x := 9
	p := &x
	if !(*p+1 == 10) {
		print("RENVO-0268 address_and_dereference_inside_expression failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
