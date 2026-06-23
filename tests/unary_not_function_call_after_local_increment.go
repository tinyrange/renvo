package main

func rtgUnaryNotCallCheck(i int) bool {
	return i == 1
}

func appMain(args []string, env []string) int {
	i := 0
	i++
	if rtgUnaryNotCallCheck(i) {
	} else {
		return 0
	}
	if !rtgUnaryNotCallCheck(i) {
		return 0
	}
	print("PASS\n")
	return 0
}
