package main

type Slice []int
type Alias = Slice
type Map map[int]int

func main() {
	s := make(Alias, 2, 4)
	s[1] = 7
	if len(s) != 2 || cap(s) != 4 || s[0] != 0 || s[1] != 7 {
		panic("slice allocation")
	}
	m := make(Map, 2)
	m[1] = 9
	if m[1] != 9 {
		panic("map allocation")
	}
	ch := make(chan int, 2)
	ch <- 11
	if cap(ch) != 2 || <-ch != 11 {
		panic("channel allocation")
	}
	items := make([]struct{ X int }, 2)
	if len(items) != 2 || items[1].X != 0 {
		panic("aggregate allocation")
	}
	{
		make := func(n int) int { return n + 1 }
		if make(2) != 3 {
			panic("shadow")
		}
	}
	println("PASS")
}
