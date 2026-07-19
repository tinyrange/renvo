package main

type renvoSelectorSmallFieldBox struct {
	b  byte
	ok bool
}

func appMain(args []string) int {
	var box renvoSelectorSmallFieldBox
	box.b = 'x'
	box.ok = true
	if box.b != 'x' {
		print("selector byte field assignment failed\n")
		return 1
	}
	if !box.ok {
		print("selector bool field assignment failed\n")
		return 1
	}
	print("PASS\n")
	return 0
}
