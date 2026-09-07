package main

type S struct{}

func f(s *S) {
	_ = s.X
}

func main() {}
