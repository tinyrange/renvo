package main

type S struct{}

func get() S { return S{} }

func main() {
	_ = get() > get()
}
