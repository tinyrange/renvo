package main

type Number int

func main() {
	value := 1
	_ = append([]Number{}, value)
}
