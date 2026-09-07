package main

type Byte byte

func main() {
	_ = append([]Byte{}, "abc"...)
}
