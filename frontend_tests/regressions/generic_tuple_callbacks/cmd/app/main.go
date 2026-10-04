package main

import "example.com/generictuplecallbacks/model"

func callback() int { return 0 }

func main() {
	f := model.Identity(model.Pair)
	n, value := f()
	if n != 5 || value != nil {
		panic("inferred tuple callback")
	}
	callback := model.Identity(f)
	n, value = callback()
	if n != 5 || value != nil {
		panic("shadowed tuple callback")
	}
	var typed func() (int, any) = model.Boxed
	typed = model.Identity(typed)
	n, value = typed()
	if n != 7 || value.(string) != "value" {
		panic("explicit tuple callback")
	}
	print("PASS\n")
}
