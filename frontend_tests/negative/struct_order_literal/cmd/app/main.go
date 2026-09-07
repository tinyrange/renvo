package main

func main() {
	_ = struct{ X int }{1} < struct{ X int }{2}
}
