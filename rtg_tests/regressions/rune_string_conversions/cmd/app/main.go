package main

type text string
type scalar rune
type scalars []scalar

func main() {
	raw := string([]byte{'A', 0xe2, 0x82, 0xac, 0x80, 0xc0, 0xaf})
	decoded := []rune(raw)
	if len(decoded) != 5 || decoded[0] != 'A' || decoded[1] != 0x20ac {
		return
	}
	for i := 2; i < len(decoded); i++ {
		if decoded[i] != 0xfffd {
			return
		}
	}
	decoded[0] = 'Z'
	if raw[0] != 'A' {
		return
	}
	encoded := string([]rune{'A', 0x20ac, -1, 0xd800, 0x110000})
	if encoded != "A€���" {
		return
	}
	named := scalars(text("A€"))
	if len(named) != 2 || named[1] != scalar(0x20ac) || text(named) != text("A€") {
		return
	}
	var nilRunes []rune
	if string(nilRunes) != "" || len([]rune("")) != 0 {
		return
	}
	print("PASS\n")
}
