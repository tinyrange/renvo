package main

type counter struct {
	step func(int) int
}

func newCounter(initial int) *counter {
	value := initial
	return &counter{step: func(delta int) int {
		value += delta
		return value
	}}
}

func join(src []byte, n int) []byte {
	var out []byte
	out = append(out, src[:n]...)
	return out
}

func main() {
	counter := newCounter(1)
	joined := join([]byte("PASS\nextra"), 5)
	if counter.step(1) != 2 || string(joined) != "PASS\n" {
		print("FAIL\n")
		return
	}
	print(string(joined))
}
