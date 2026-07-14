package main

type F func(int) int
type (
	G func(string)
	H = func() bool
)
func main() { type Local func() int }
