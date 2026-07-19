package main

var renvo0694Value int = 2

func renvo0694Add() {
	renvo0694Value = renvo0694Value + 5
}

func appMain(args []string) int {
	renvo0694Add()
	if renvo0694Value != 7 {
		print("RENVO-0694 helper global mutation failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
