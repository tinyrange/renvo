package main

type scorer interface {
	score() int
}

type item struct {
	value int
}

func (i item) score() int {
	return i.value + 1
}

func check(s scorer) bool {
	return s.score() == 5
}

func main() {
	corpusOK := check(item{value: 4})
	if !corpusOK {

		print("FAIL\n")
		return
	}
	print("PASS\n")

}
