package main

type Renvo0602Box struct{ value int }

func appMain(args []string) int {
	box := Renvo0602Box{value: 12}
	if box.value != 12 {
		print("RENVO-0602 one field failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
