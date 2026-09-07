package main

type callback func(int) int
type holder struct { callback callback }
func callField(h holder) int { return h.callback(1) }

func main() {
	n := 3
	f := callback(func(x int) int { return n+x })
	if f(4) != 7 { panic("converted closure") }
	if callField(holder{callback:f}) != 4 { panic("callback field named like type") }
	n = 5
	if f(4) != 9 { panic("shared capture") }
	callback := f
	if callback(2) != 7 { panic("shadowed type name") }
	print("PASS\n")
}
