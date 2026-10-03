package main

type compareText string
type compareBytes []byte

func compareMutate(data []byte) string {
	data[0] = 'X'
	return "abc"
}

func appMain() int {
	data := []byte{'a', 'b', 'c'}
	want := "abc"
	if string(data) != want || want != string(data) || string(data[1:]) != "bc" {
		panic("temporary comparison")
	}
	named := compareBytes{'a', 'b', 'c'}
	if compareText(named) != compareText("abc") {
		panic("named temporary comparison")
	}
	var empty []byte
	if string(empty) != "" || string(empty) == "abc" {
		panic("empty temporary comparison")
	}
	saved := string(data)
	if string(data) != compareMutate(data) {
		panic("comparison operand ordering")
	}
	if saved != "abc" || string(data) != "Xbc" {
		panic("stored string ownership")
	}
	print("PASS\n")
	return 0
}
