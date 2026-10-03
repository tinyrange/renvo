package main

type indexFrameRecord struct {
	a int
	b int
	c int
}
type indexFrameWide struct{ values [20]int }

func indexFrameSource(values []indexFrameRecord, index *int) []indexFrameRecord {
	*index = 2
	return values
}

func appMain(args []string) int {
	values := []indexFrameRecord{{11, 12, 13}, {21, 22, 23}, {31, 32, 33}}
	index := 0
	got := indexFrameSource(values, &index)[index]
	if index != 2 || got.a != 31 || got.b != 32 || got.c != 33 {
		return 1
	}
	values[index].b = 42
	if values[2].b != 42 {
		return 2
	}
	var wide [3]indexFrameWide
	wide[index].values[19] = 91
	if wide[2].values[19] != 91 {
		return 3
	}
	short := []uint16{3, 5, 7}
	words := []uint32{11, 13, 17}
	if short[index] != 7 || words[index] != 17 {
		return 4
	}
	var empty [3]struct{}
	_ = empty[index]
	print("PASS\n")
	return 0
}
