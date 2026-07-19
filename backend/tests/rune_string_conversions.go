package main

type runeConversionString string
type runeConversionRune rune
type runeConversionRunes []runeConversionRune

func appMain() int {
	raw := string([]byte{
		'A', 0xe2, 0x82, 0xac,
		0x80, 0xc0, 0xaf,
		0xe0, 0x80, 0x80,
		0xed, 0xa0, 0x80,
		0xf4, 0x90, 0x80, 0x80,
		0xf0, 0x9f,
	})
	decoded := []rune(raw)
	want := []rune{
		'A', 0x20ac,
		0xfffd, 0xfffd, 0xfffd,
		0xfffd, 0xfffd, 0xfffd,
		0xfffd, 0xfffd, 0xfffd,
		0xfffd, 0xfffd, 0xfffd, 0xfffd,
		0xfffd, 0xfffd,
	}
	if len(decoded) != len(want) {
		print("FAIL decode length\n")
		return 1
	}
	for i := 0; i < len(want); i++ {
		if decoded[i] != want[i] {
			print("FAIL decode value\n")
			return 1
		}
	}
	decoded[0] = 'Z'
	if raw[0] != 'A' {
		print("FAIL independent backing\n")
		return 1
	}

	encoded := string([]rune{'A', 0x7f, 0x80, 0x7ff, 0x800, 0x20ac, 0xffff, 0x10000, 0x10ffff, -1, 0xd800, 0x110000})
	wantBytes := []byte{
		65, 127, 194, 128, 223, 191, 224, 160, 128, 226, 130, 172,
		239, 191, 191, 240, 144, 128, 128, 244, 143, 191, 191,
		239, 191, 189, 239, 191, 189, 239, 191, 189,
	}
	if len(encoded) != len(wantBytes) {
		print("FAIL encode length\n")
		return 1
	}
	for i := 0; i < len(wantBytes); i++ {
		if encoded[i] != wantBytes[i] {
			print("FAIL encode value\n")
			return 1
		}
	}

	named := runeConversionRunes(runeConversionString("A€𐀀"))
	if len(named) != 3 || named[0] != runeConversionRune('A') || named[1] != runeConversionRune(0x20ac) || named[2] != runeConversionRune(0x10000) {
		print("FAIL named decode\n")
		return 1
	}
	if runeConversionString(named) != runeConversionString("A€𐀀") {
		print("FAIL named encode\n")
		return 1
	}

	var nilRunes []rune
	if string(nilRunes) != "" || len([]rune("")) != 0 {
		print("FAIL empty conversion\n")
		return 1
	}
	print("PASS\n")
	return 0
}
