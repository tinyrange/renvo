package main

type Text string

func main() {
	var text Text = "abc"
	array := [3]int{1, 2, 3}
	slice := make([]int, 2, 4)
	mapping := map[int]int{1: 2}
	channel := make(chan int, 2)
	if len(text) != 3 || len(array) != 3 || cap(&array) != 3 || len(slice) != 2 || cap(slice) != 4 || len(mapping) != 1 || len(channel) != 0 || cap(channel) != 2 {
		panic("length or capacity")
	}
	value := 1
	{
		value := "abcd"
		if len(value) != 4 {
			panic("scope")
		}
	}
	len := func(n int) int { return n + 1 }
	if len(value) != 2 {
		panic("shadowed len")
	}
	println("PASS")
}
