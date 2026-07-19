package main

type renvo0671Level int

var renvo0671Global renvo0671Level = renvo0671Level(3)

func appMain(args []string) int {
	renvo0671Global = renvo0671Global + renvo0671Level(4)
	if int(renvo0671Global) != 7 {
		print("RENVO-0671 named global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
