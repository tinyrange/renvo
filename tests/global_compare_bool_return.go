package main

const wantOS = 2

var currentOS int = 1

func isWanted() bool {
	return currentOS == wantOS
}

func appMain() int {
	if !isWanted() {
		print("PASS\n")
	}
	return 0
}
