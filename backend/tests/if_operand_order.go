package main

var order int

func first() int  { order = order*10 + 1; return 3 }
func second() int { order = order*10 + 2; return 4 }

func appMain(args []string) int {
	if first() < second() {
	} else {
		return 1
	}
	if order != 12 {
		return 2
	}
	order = 0
	if first() == second() {
		return 3
	}
	if order != 12 {
		return 4
	}
	order = 0
	if (first() == 3) == (second() == 4) {
	} else {
		return 5
	}
	if order != 12 {
		return 6
	}
	print("PASS\n")
	return 0
}
