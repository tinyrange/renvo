package main

func main() {
	const (
		_ = iota + 254
		a
		b
	)
	_ = append([]byte{}, b)
}
