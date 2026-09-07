package main

import "fmt"

const outer = 256

func main() {
	const outer = outer - 1
	const saved = outer
	const (
		a, b = iota + 253, iota + 254
		c, d
	)
	const reset = iota
	values := append([]byte{}, outer, a, b, c, d, reset)
	{
		const outer = 1
		values = append(values, saved, outer)
	}
	if len(values) != 8 || values[0] != 255 || values[1] != 253 || values[2] != 254 || values[3] != 254 || values[4] != 255 || values[5] != 0 || values[6] != 255 || values[7] != 1 {
		panic("local constant bounds")
	}
	fmt.Println("PASS")
}
