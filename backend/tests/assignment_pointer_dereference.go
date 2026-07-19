package main

func appMain(args []string) int {
	x := 2
	p := &x
	*p = 8
	if x != 8 {
		print("RENVO-0341 pointer assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
