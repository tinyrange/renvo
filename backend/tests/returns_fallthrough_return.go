package main

var renvo0527Hit int

func renvo0527Mark() {
	renvo0527Hit = 7
}

func appMain(args []string) int {
	renvo0527Mark()
	if renvo0527Hit != 7 {
		print("RENVO-0527 fallthrough return failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
