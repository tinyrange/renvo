package main

func main() {
	const count = 2
	var (
		first  struct{ Values [count]int }
		second struct{ Values [count]int }
	)
	var pointer *struct{ Values [count]int }
	var array [count]struct{ Values [count]int }
	var slice []struct{ Values [count]int }
	first.Values[1] = 3
	second.Values[1] = 4
	pointer = &first
	array[1] = second
	slice = append(slice, first)
	if pointer.Values[1] != 3 || array[1].Values[1] != 4 || len(slice) != 1 || slice[0].Values[1] != 3 {
		panic("local anonymous wrappers")
	}
	println("PASS")
}
