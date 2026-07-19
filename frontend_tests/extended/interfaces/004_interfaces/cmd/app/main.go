package main

type scorer interface {
	score() int
}

type item struct {
	value int
}

func (i item) score() int {
	return i.value + 4
}

func check(s scorer) bool {
	return s.score() == 11
}

func main() {
	corpusOK := false
	if check(item{value: 7}) {
		corpusOK = true
	}
	if corpusOK {
		print("PASS\n")
		return
	}

	print("FAIL\n")
}
