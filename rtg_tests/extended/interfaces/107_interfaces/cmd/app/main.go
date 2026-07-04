package main

type scorer interface {
	score() int
}

type item struct {
	value int
}

func (i item) score() int {
	return i.value + 8
}

func check(s scorer) bool {
	return s.score() == 19
}

func main() {
	if check(item{value: 11}) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
