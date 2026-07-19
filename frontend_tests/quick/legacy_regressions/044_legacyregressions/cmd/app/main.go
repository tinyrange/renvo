package main

type issue44S struct{ x int }

func (s issue44S) Bump() int { s.x++; return s.x }

func main() {
	s := issue44S{x: 3}
	if s.Bump() == 4 && s.x == 3 {
		print("PASS\n")
		return
	}
	print("FAIL\n")
}
