package main

type renvo0698Named int

var renvo0698Value renvo0698Named = renvo0698Named(14)

func appMain(args []string) int {
	if int(renvo0698Value) != 14 {
		print("RENVO-0698 named global failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
