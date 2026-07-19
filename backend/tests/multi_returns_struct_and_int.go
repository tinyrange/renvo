package main

type renvo1011Cell struct {
	value int
	tag   byte
}

func renvo1011Make() (renvo1011Cell, int) {
	return renvo1011Cell{value: 8, tag: 'q'}, 3
}

func appMain(args []string) int {
	cell, status := renvo1011Make()
	if cell.value+status != 11 || cell.tag != 'q' {
		print("RENVO-1011 struct and int returns failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
