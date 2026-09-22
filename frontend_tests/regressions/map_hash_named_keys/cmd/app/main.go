package main

import "strconv"

type Number int64
type NumberAlias = Number
type Text string
type TextAlias = Text

func main() {
	numbers := make(map[NumberAlias]int)
	texts := make(map[TextAlias]int)
	for i := 0; i < 100000; i++ {
		numbers[Number(i)*17-50000] = i + 1
		texts[Text("key/"+strconv.Itoa(i))] = i + 1
	}
	for i := 0; i < 100000; i += 2 {
		delete(numbers, Number(i)*17-50000)
		delete(texts, Text("key/"+strconv.Itoa(i)))
	}
	for i := 0; i < 100000; i++ {
		n, nok := numbers[Number(i)*17-50000]
		s, sok := texts[Text("key/"+strconv.Itoa(i))]
		if i%2 == 0 {
			if nok || sok {
				panic("deleted key")
			}
		} else if !nok || !sok || n != i+1 || s != i+1 {
			panic("moved entry or unequal string storage")
		}
	}
	if len(numbers) != 50000 || len(texts) != 50000 {
		panic("length")
	}
	for i := 0; i < 100000; i += 2 {
		numbers[Number(i)*17-50000] = -i
		texts[Text("key/"+strconv.Itoa(i))] = -i
	}
	if len(numbers) != 100000 || len(texts) != 100000 {
		panic("reinsertion")
	}
	clear(numbers)
	clear(texts)
	if len(numbers) != 0 || len(texts) != 0 {
		panic("clear")
	}
	numbers[-1] = 42
	texts[""] = 1
	texts["a\x00b"] = 2
	texts["é界"] = 3
	texts["a"] = 4
	delete(texts, "a")
	if numbers[-1] != 42 || texts[""] != 1 || texts["a\x00b"] != 2 || texts["é界"] != 3 {
		panic("reuse or byte hashing")
	}
	var nilMap map[Text]int
	delete(nilMap, "missing")
	clear(nilMap)
	if _, ok := nilMap[""]; ok {
		panic("nil lookup")
	}
	println("PASS")
}
