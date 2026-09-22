package main

const value = 1

var storage int
var values = []int{1}

func pointer() *int { return &storage }
func slice() []int  { return values }

func main() {
	value := 2
	value += 1
	{
		const value = 7
		if value != 7 {
			panic("constant")
		}
	}
	value++
	*pointer() = value
	slice()[0] = value + 1
	if storage != 4 || values[0] != 5 {
		panic("writable target")
	}
	if value := 2; value > 0 {
		value = 3
		if value != 3 {
			panic("header scope")
		}
	}
	println("PASS")
}
