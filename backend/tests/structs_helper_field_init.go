package main

type Renvo0623Box struct{ value int }

func renvo0623Value() int {
	return 31
}

func appMain(args []string) int {
	box := Renvo0623Box{value: renvo0623Value()}
	if box.value != 31 {
		print("RENVO-0623 helper field init failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
