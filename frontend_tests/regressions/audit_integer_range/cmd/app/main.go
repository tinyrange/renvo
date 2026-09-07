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
	println("PASS")
}
