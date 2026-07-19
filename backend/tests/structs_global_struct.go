package main

const renvo0621Want = 23

type Renvo0621Box struct{ value int }

var renvo0621Global = Renvo0621Box{value: renvo0621Want}

func appMain(args []string) int {
	if renvo0621Global.value != 23 {
		print("RENVO-0621 global struct failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
