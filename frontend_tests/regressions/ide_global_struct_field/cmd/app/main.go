package main

type Color struct {
	R byte
	G byte
	B byte
	A byte
}

var white = Color{R: 255, G: 255, B: 255, A: 255}

type Widget struct {
	visible    bool
	background Color
}

func newWidget() *Widget {
	return &Widget{visible: true, background: white}
}

func main() {
	widget := newWidget()
	if !widget.visible || widget.background.R != 255 || widget.background.G != 255 || widget.background.B != 255 || widget.background.A != 255 {
		print("FAIL\n")
		return
	}
	print("PASS\n")
	return
}
