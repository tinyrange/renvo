package main

import (
	"renvo.dev/std/graphics"
	"renvo.dev/std/graphics/gofont"
)

func main() {
	font := gofont.NewMono(15)
	if font == nil {
		print("FAIL font nil\n")
		return
	}
	wide := graphics.MeasureText(font, "WW").Width
	narrow := graphics.MeasureText(font, "ii").Width
	if wide <= 0 || wide != narrow {
		print("FAIL metrics\n")
		return
	}
	print("PASS\n")
}
