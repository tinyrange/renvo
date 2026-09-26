package main

func main() {
	const (
		_ = iota
		index
	)
	_ = [1]int{index: 7}
}
