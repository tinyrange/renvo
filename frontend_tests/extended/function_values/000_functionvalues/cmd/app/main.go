package main

func add(a int, b int) int                          { return a + b }
func mark(trace *int, step int) int                 { *trace = *trace*10 + step; return step }
func apply(fn func(int, int) int, a int, b int) int { return fn(a, b) }

func main() {
	trace := 0
	fn := add
	left := mark(&trace, 2)
	right := mark(&trace, 3)
	got := apply(fn, left, right)
	if trace == 23 && got == 5 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
