package main

type Box struct {
	words    []int
	negative bool
	ok       bool
}

func makeBox(value int) Box { out := Box{ok: true}; out.words = append(out.words, value); return out }
func shiftBox(v Box, count int, left bool) Box {
	if !v.ok || count < 0 {
		return Box{}
	}
	if left {
		return Box{words: []int{v.words[0] << uint(count)}, ok: true}
	}
	return v
}
func evaluate(v Box, name string) bool {
	bits := 64
	if name == "uint8" {
		bits = 8
	}
	b := shiftBox(makeBox(1), bits, true)
	comparison := b.words[0]
	return b.ok && len(b.words) == 1 && comparison == 256
}
func appMain() int {
	if !evaluate(makeBox(32), "uint8") {
		return 1
	}
	print("PASS\n")
	return 0
}
