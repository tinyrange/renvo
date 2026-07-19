package main

func renvoUnaryNotCallCheck(i int) bool {
	return i == 1
}

func appMain(args []string, env []string) int {
	i := 0
	i++
	if renvoUnaryNotCallCheck(i) {
	} else {
		return 0
	}
	if !renvoUnaryNotCallCheck(i) {
		return 0
	}
	print("PASS\n")
	return 0
}
