package main

func main() {
	if modern()+legacy() != 42 {
		panic("release selection")
	}
	println("PASS")
}
