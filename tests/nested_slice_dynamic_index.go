package main

type nestedSliceField struct {
	value int
}

func nestedSliceFind(groups [][]nestedSliceField) int {
	total := 0
	for i := 0; i < len(groups); i++ {
		total += len(groups[i])
		if cap(groups[i]) < len(groups[i]) {
			return -1
		}
		for j := 0; j < len(groups[i]); j++ {
			item := groups[i][j]
			if item.value == 7 {
				return total + item.value
			}
		}
	}
	return 0
}

func appMain() int {
	first := []nestedSliceField{{value: 1}, {value: 7}}
	second := []nestedSliceField{{value: 9}}
	groups := [][]nestedSliceField{first, second}
	if nestedSliceFind(groups) != 9 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
