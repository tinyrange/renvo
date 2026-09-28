package main

type detail struct{ Message string }
type outer struct{ Value *detail }

func main() {
	x := outer{&detail{Message: "nested"}}
	if x.Value.Message != "nested" {
		panic("nested positional literal")
	}
	print("PASS\n")
}
