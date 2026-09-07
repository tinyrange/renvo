package main

func main() {
	var x struct{ Field int }
	_ = x..Field
}
