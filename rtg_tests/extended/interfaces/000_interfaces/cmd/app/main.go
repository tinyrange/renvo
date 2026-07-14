package main

type scorer interface { score() int }
type item struct { value int }

func mark(trace *int, step int) int { *trace = *trace*10+step; return 7 }
func (i item) score() int { return i.value }

func main() {
	trace := 0
	value := item{value: mark(&trace, 1)}
	var dynamic scorer = value
	got := dynamic.score()
	if trace == 1 && got == 7 { print("PASS\n"); return }
	print("FAIL\n")
}
