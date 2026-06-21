package main

type rtg1011Cell struct {
	value int
	tag   byte
}

func rtg1011Make() (rtg1011Cell, int) {
	return rtg1011Cell{value: 8, tag: 'q'}, 3
}

func appMain(args []string) int {
	cell, status := rtg1011Make()
	if cell.value+status != 11 || cell.tag != 'q' {
		print("RTG-1011 struct and int returns failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
