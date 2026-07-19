package main

type Renvo0622Node struct{ value int }

var renvo0622Start int = 2

func appMain(args []string) int {
	Renvo0622Node := renvo0622Start + 5
	if Renvo0622Node != 7 {
		print("RENVO-0622 shadow local failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
