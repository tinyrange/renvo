package main

func emit(s string) {
	write(1, []byte(s), -1)
}

func appMain() int {
	emit("PASS\n")
	return 0
}
