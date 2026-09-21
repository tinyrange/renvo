package main

var count = 9

func main() {
	const count = 2
	const true = 3
	const nil = 4
	var a [count]int
	var b [true]int
	var c [nil]int
	{
		count := 8
		_ = count
	}
	var d [count]int
	if len(a) != 2 || len(b) != 3 || len(c) != 4 || len(d) != 2 {
		panic("array length binding kinds")
	}
	println("PASS")
}
