package main

func discardedPair() (int, int) {
	return 12, 34
}

func appMain() int {
	discardedPair()
	print("PASS\n")
	return 0
}
