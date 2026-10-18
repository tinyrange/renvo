package main

const factor = 4096
const alias = factor
const combined = (alias + 3) ^ 3
const (
	first = iota
	second
)
const typed = uint64(7)

type Counter uint64

func check(value uint64, signed int64, counter Counter) bool {
	a := value * factor
	b := alias * value
	c := signed * combined
	d := combined * signed
	e := counter + factor
	f := factor + counter
	g := value * (second + 1)
	h := value + typed
	return a == uint64(1)<<44 && b == a && c == -int64(1)<<44 && d == c &&
		e == Counter(1)<<32+4096 && f == e && g == uint64(1)<<33 && h == value+7
}
func appMain() int {
	if !check(uint64(1)<<32, -int64(1)<<32, Counter(1)<<32) {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
