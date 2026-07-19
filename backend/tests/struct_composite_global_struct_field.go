package main

type compositeGlobalColor struct {
	r byte
	g byte
	b byte
	a byte
}

var compositeGlobalWhite = compositeGlobalColor{r: 255, g: 255, b: 255, a: 255}

type compositeGlobalWidget struct {
	visible bool
	color   compositeGlobalColor
}

func compositeGlobalNewWidget() *compositeGlobalWidget {
	return &compositeGlobalWidget{visible: true, color: compositeGlobalWhite}
}

func appMain() int {
	widget := compositeGlobalNewWidget()
	if !widget.visible || widget.color.r != 255 || widget.color.g != 255 || widget.color.b != 255 || widget.color.a != 255 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
