package main

type Text string
type Slice []int

func main() {
	var text Text = "abc"
	values := Slice{1, 2}
	array := [2]int{3, 4}
	pointer := &array
	mapping := map[int]int{1: 5}
	if text[1] != 'b' || values[0] != 1 || array[1] != 4 || pointer[0] != 3 || mapping[1] != 5 {
		panic("index base")
	}
	v := 2
	{
		v := []int{6}
		if v[0] != 6 {
			panic("scope")
		}
	}
	if v != 2 {
		panic("outer binding")
	}
	println("PASS")
}
