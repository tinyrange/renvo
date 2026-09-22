package main

var calls int

func bound() int { calls++; return 5 }
func main() {
	total := 0
	for i := range bound() {
		if i == 2 {
			continue
		}
		total += i
	}
	if total != 8 || calls != 1 {
		panic("bound or continue")
	}
	x := 19
	for x = range -2 {
		panic("negative")
	}
	if x != 19 {
		panic("empty range assignment")
	}
	for x = range 3 {
	}
	if x != 2 {
		panic("assignment")
	}
	for range 3 {
		total++
	}
	for _ = range uint8(2) {
		total++
	}
	for i := range 2 {
		for j := range 2 {
			total += i + j
		}
	}
	if total != 17 {
		panic("nested")
	}
	__renvo_integer_range_index_0 := 17
	__renvo_integer_range_bound_0 := 19
	var addresses [3]*int
	for i := range 3 {
		addresses[i] = &i
		if __renvo_integer_range_index_0 != 17 || __renvo_integer_range_bound_0 != 19 {
			panic("generated variable capture")
		}
	}
	for i := range 3 {
		if __renvo_integer_range_index_0 != 17 || __renvo_integer_range_bound_0 != 19 {
			panic("generated variable capture in final loop")
		}
		if *addresses[i] != i {
			panic("iteration binding")
		}
	}
	println("PASS")
}
