package main

var order int

func touch(n int) int { order = order*10 + n; return n }

var _ = touch(1)
var _ = touch(2)
var _ = "unused value with a different type"
var _, kept = touch(3), touch(4)

const _ = true
const _ = 2

var _ = touch

func main() {
	if order != 1234 || kept != 4 {
		panic("blank initializer effects or declaration order lost")
	}
	println("PASS")
}
