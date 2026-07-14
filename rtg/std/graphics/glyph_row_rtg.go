//go:build rtg

package graphics

func packGlyph(a, b, c, d, e, f, g int) int {
	return a<<30 | b<<25 | c<<20 | d<<15 | e<<10 | f<<5 | g
}

func glyphBits(r int) int {
	if r >= 'a' && r <= 'z' {
		r -= 'a' - 'A'
	}
	switch r {
	case 'A':
		return packGlyph(14, 17, 17, 31, 17, 17, 17)
	case 'B':
		return packGlyph(30, 17, 17, 30, 17, 17, 30)
	case 'C':
		return packGlyph(14, 17, 16, 16, 16, 17, 14)
	case 'D':
		return packGlyph(30, 17, 17, 17, 17, 17, 30)
	case 'E':
		return packGlyph(31, 16, 16, 30, 16, 16, 31)
	case 'F':
		return packGlyph(31, 16, 16, 30, 16, 16, 16)
	case 'G':
		return packGlyph(14, 17, 16, 23, 17, 17, 15)
	case 'H':
		return packGlyph(17, 17, 17, 31, 17, 17, 17)
	case 'I':
		return packGlyph(14, 4, 4, 4, 4, 4, 14)
	case 'J':
		return packGlyph(7, 2, 2, 2, 18, 18, 12)
	case 'K':
		return packGlyph(17, 18, 20, 24, 20, 18, 17)
	case 'L':
		return packGlyph(16, 16, 16, 16, 16, 16, 31)
	case 'M':
		return packGlyph(17, 27, 21, 21, 17, 17, 17)
	case 'N':
		return packGlyph(17, 25, 21, 19, 17, 17, 17)
	case 'O':
		return packGlyph(14, 17, 17, 17, 17, 17, 14)
	case 'P':
		return packGlyph(30, 17, 17, 30, 16, 16, 16)
	case 'Q':
		return packGlyph(14, 17, 17, 17, 21, 18, 13)
	case 'R':
		return packGlyph(30, 17, 17, 30, 20, 18, 17)
	case 'S':
		return packGlyph(15, 16, 16, 14, 1, 1, 30)
	case 'T':
		return packGlyph(31, 4, 4, 4, 4, 4, 4)
	case 'U':
		return packGlyph(17, 17, 17, 17, 17, 17, 14)
	case 'V':
		return packGlyph(17, 17, 17, 17, 17, 10, 4)
	case 'W':
		return packGlyph(17, 17, 17, 21, 21, 21, 10)
	case 'X':
		return packGlyph(17, 17, 10, 4, 10, 17, 17)
	case 'Y':
		return packGlyph(17, 17, 10, 4, 4, 4, 4)
	case 'Z':
		return packGlyph(31, 1, 2, 4, 8, 16, 31)
	case '0':
		return packGlyph(14, 17, 19, 21, 25, 17, 14)
	case '1':
		return packGlyph(4, 12, 4, 4, 4, 4, 14)
	case '2':
		return packGlyph(14, 17, 1, 2, 4, 8, 31)
	case '3':
		return packGlyph(30, 1, 1, 14, 1, 1, 30)
	case '4':
		return packGlyph(2, 6, 10, 18, 31, 2, 2)
	case '5':
		return packGlyph(31, 16, 16, 30, 1, 1, 30)
	case '6':
		return packGlyph(14, 16, 16, 30, 17, 17, 14)
	case '7':
		return packGlyph(31, 1, 2, 4, 8, 8, 8)
	case '8':
		return packGlyph(14, 17, 17, 14, 17, 17, 14)
	case '9':
		return packGlyph(14, 17, 17, 15, 1, 1, 14)
	case ' ':
		return 0
	case '.':
		return packGlyph(0, 0, 0, 0, 0, 6, 6)
	case ',':
		return packGlyph(0, 0, 0, 0, 6, 6, 4)
	case ':':
		return packGlyph(0, 6, 6, 0, 6, 6, 0)
	case ';':
		return packGlyph(0, 6, 6, 0, 6, 6, 4)
	case '!':
		return packGlyph(4, 4, 4, 4, 4, 0, 4)
	case '?':
		return packGlyph(14, 17, 1, 2, 4, 0, 4)
	case '-':
		return packGlyph(0, 0, 0, 31, 0, 0, 0)
	case '_':
		return packGlyph(0, 0, 0, 0, 0, 0, 31)
	case '+':
		return packGlyph(0, 4, 4, 31, 4, 4, 0)
	case '/':
		return packGlyph(1, 2, 2, 4, 8, 8, 16)
	case '\\':
		return packGlyph(16, 8, 8, 4, 2, 2, 1)
	case '(':
		return packGlyph(2, 4, 8, 8, 8, 4, 2)
	case ')':
		return packGlyph(8, 4, 2, 2, 2, 4, 8)
	case '[':
		return packGlyph(14, 8, 8, 8, 8, 8, 14)
	case ']':
		return packGlyph(14, 2, 2, 2, 2, 2, 14)
	case '=':
		return packGlyph(0, 0, 31, 0, 31, 0, 0)
	}
	return packGlyph(31, 17, 5, 4, 4, 0, 4)
}

func glyphRow(r, y int) byte {
	return byte((glyphBits(r) >> uint((6-y)*5)) & 31)
}
