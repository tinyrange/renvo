package main

type Counter struct {
	Step func(delta int) int
}

func newCounter(initial int) *Counter {
	value := initial
	return &Counter{
		Step: func(delta int) int {
			value += delta
			return value
		},
	}
}

func main() {
	counter := newCounter(10)
	if counter.Step == nil || counter.Step(12) != 22 || counter.Step(20) != 42 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
	return
}
