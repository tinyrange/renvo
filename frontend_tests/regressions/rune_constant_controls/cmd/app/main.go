package main

const Letter = '界'

var values ['b' - 'a']int

func main() {
	a := append([]byte{}, '\xff', '\377', '\'', '\n')
	if a[0] != 255 || a[1] != 255 || a[2] != 39 || a[3] != 10 {
		panic("byte escapes")
	}
	b := append([]rune{}, Letter, '\u754c', '\U0001f600')
	if b[0] != 30028 || b[1] != 30028 || b[2] != 128512 {
		panic("unicode")
	}
	c := append([]byte{}, Letter-Letter+'a')
	if c[0] != 97 || len(values) != 1 {
		panic("arithmetic")
	}
	println("PASS")
}
