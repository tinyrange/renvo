package main

func renvoStringBytesArgEmit(s string) {
	write(1, []byte(s), -1)
}

func appMain(args []string) int {
	renvoStringBytesArgEmit("PASS\n")
	return 0
}
