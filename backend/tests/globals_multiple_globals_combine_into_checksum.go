package main

const renvo0700A = 10

var renvo0700B int = 20
var renvo0700C byte = 3

func appMain(args []string) int {
	if renvo0700A+renvo0700B+int(renvo0700C) != 33 {
		print("RENVO-0700 global checksum failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
