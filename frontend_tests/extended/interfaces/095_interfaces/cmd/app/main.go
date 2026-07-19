package main

type scorer interface {
	score() int
}

type item struct {
	value int
}

func (i item) score() int {
	return i.value + 5
}

func check(s scorer) bool {
	return s.score() == 15
}

func main() {
	if check(item{value: 10}) {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
