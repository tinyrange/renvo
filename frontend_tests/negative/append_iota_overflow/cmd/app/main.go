package main

const (
	A = iota + 255
	B
)

func main() {
	_ = append([]byte{}, B)
}
