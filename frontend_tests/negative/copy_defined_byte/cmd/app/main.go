package main

type Byte byte

func main() {
	_ = copy([]Byte{}, "abc")
}
