package main

type pointerCallGlyph struct {
	advance float64
}

type pointerCallFont struct{}

func (font *pointerCallFont) cachedGlyph() *pointerCallGlyph {
	return &pointerCallGlyph{advance: 2}
}

func appMain(args []string) int {
	font := &pointerCallFont{}
	x := 1.0
	x += font.cachedGlyph().advance * 4
	if x != 9.0 {
		print("FAIL\n")
		return 1
	}
	print("PASS\n")
	return 0
}
