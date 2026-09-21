package main

type Slice []int

func main() {
	var values Slice
	values = append(values, 1, 2)
	values = append(values, []int{3, 4}...)
	values = append(values, nil...)
	values = append(values)
	if len(values) != 4 || values[0] != 1 || values[3] != 4 {
		panic("slice append")
	}
	bytes := append([]byte{}, "abc"...)
	if string(bytes) != "abc" {
		panic("string expansion")
	}
	n := 2
	{
		n := []int{1}
		n = append(n, 3)
		if len(n) != 2 || n[1] != 3 {
			panic("scope")
		}
	}
	append := func(a, b int) int { return a + b }
	if append(n, 3) != 5 {
		panic("shadowed append")
	}
	println("PASS")
}
