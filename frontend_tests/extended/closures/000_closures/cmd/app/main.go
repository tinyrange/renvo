package main

func mark(trace *int, step int) int    { *trace = *trace*10 + step; return step }
func makeAdder(base int) func(int) int { return func(value int) int { return base + value } }

func main() {
	trace := 0
	next := makeAdder(mark(&trace, 1))
	got := next(mark(&trace, 2))
	if trace == 12 && got == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
