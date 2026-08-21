package main

type escapedMakeBox struct {
	values []rune
}

func makeEscapedBox(value rune) *escapedMakeBox {
	box := &escapedMakeBox{}
	values := make([]rune, 2)
	values[0] = value
	values[1] = value
	box.values = values
	return box
}

func assignExistingSlice(box *escapedMakeBox, values []rune) {
	box.values = values
}

func appMain(args []string) int {
	first := makeEscapedBox('a')
	second := makeEscapedBox('c')
	third := makeEscapedBox('o')
	fourth := makeEscapedBox('m')
	if first.values[0] != 'a' || second.values[0] != 'c' || third.values[0] != 'o' || fourth.values[0] != 'm' {
		print("make slice struct field backing was reused\n")
		return 1
	}

	values := make([]rune, 1)
	values[0] = 'x'
	box := &escapedMakeBox{}
	assignExistingSlice(box, values)
	values[0] = 'y'
	if box.values[0] != 'y' {
		print("stable slice assignment lost descriptor aliasing\n")
		return 1
	}

	print("PASS\n")
	return 0
}
