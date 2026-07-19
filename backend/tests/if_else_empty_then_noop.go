package main

func renvo0371Noop() {}
func appMain(args []string) int {
	x := 1
	if x == 1 {
		renvo0371Noop()
	} else {
		x = 9
	}
	if x != 1 {
		print("RENVO-0371 noop then failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
