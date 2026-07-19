package main

type slicePair struct {
	first  []int
	second []int
}

func makeSlicePair(first []int, second []int) slicePair {
	return slicePair{first: first, second: second}
}

func appMain() int {
	first := []int{11, 12}
	second := []int{21, 22}
	pair := makeSlicePair(first, second)
	if len(pair.first) != 2 || len(pair.second) != 2 || pair.first[1] != 12 || pair.second[1] != 22 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
