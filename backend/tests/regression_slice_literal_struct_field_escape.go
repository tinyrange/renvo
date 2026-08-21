package main

type escapedLiteralBox struct {
	value  int
	values []int
}

func setEscapedLiteral(box *escapedLiteralBox, value int) {
	box.value = value
	box.values = []int{value}
}

func returnEscapedLiteral(value int) escapedLiteralBox {
	return escapedLiteralBox{value: value, values: []int{value}}
}

func appMain(args []string) int {
	var first escapedLiteralBox
	var second escapedLiteralBox
	setEscapedLiteral(&first, 1)
	setEscapedLiteral(&second, 2)
	if first.value != 1 || first.values[0] != 1 || second.value != 2 || second.values[0] != 2 {
		print("slice literal pointer field backing was reused\n")
		return 1
	}

	third := returnEscapedLiteral(3)
	fourth := returnEscapedLiteral(4)
	if third.value != 3 || third.values[0] != 3 || fourth.value != 4 || fourth.values[0] != 4 {
		print("slice literal struct return backing was reused\n")
		return 1
	}

	print("PASS\n")
	return 0
}
