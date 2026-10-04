package main

type storageAliasBox struct {
	values []int
}

func storageAliasField(box *storageAliasBox) []int { return box.values }
func storageAliasIndex(values [][]int) []int       { return values[0] }
func storageAliasPointer(values *[]int) []int      { return *values }
func storageAliasView(box *storageAliasBox) []int  { return box.values[1:3:4] }
func storageAliasForward(box *storageAliasBox) []int {
	values := storageAliasField(box)
	return values
}

func appMain() int {
	values := make([]int, 3, 4)
	box := storageAliasBox{values: values}
	field := storageAliasField(&box)
	indexed := storageAliasIndex([][]int{values})
	pointer := storageAliasPointer(&values)
	forward := storageAliasForward(&box)
	view := storageAliasView(&box)
	values[0] = 17
	if field[0] != 17 || indexed[0] != 17 || pointer[0] != 17 || forward[0] != 17 {
		panic("returned slice storage identity")
	}
	view[0] = 23
	if values[1] != 23 || len(view) != 2 || cap(view) != 3 || cap(field) != 4 {
		panic("returned slice view identity")
	}
	field = append(field, 31)
	if values[:4][3] != 31 {
		panic("returned slice capacity identity")
	}
	print("PASS\n")
	return 0
}
