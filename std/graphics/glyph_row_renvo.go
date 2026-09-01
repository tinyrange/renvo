//go:build renvo

package graphics

func packGlyph(a, b, c, d, e, f, g, row int) byte {
	if row == 0 {
		return byte(a)
	}
	if row == 1 {
		return byte(b)
	}
	if row == 2 {
		return byte(c)
	}
	if row == 3 {
		return byte(d)
	}
	if row == 4 {
		return byte(e)
	}
	if row == 5 {
		return byte(f)
	}
	return byte(g)
}

func glyphBits(r, y int) byte {
	switch r {
	case 'A':
		return packGlyph(14, 17, 17, 31, 17, 17, 17, y)
	case 'B':
		return packGlyph(30, 17, 17, 30, 17, 17, 30, y)
	case 'C':
		return packGlyph(14, 17, 16, 16, 16, 17, 14, y)
	case 'D':
		return packGlyph(30, 17, 17, 17, 17, 17, 30, y)
	case 'E':
		return packGlyph(31, 16, 16, 30, 16, 16, 31, y)
	case 'F':
		return packGlyph(31, 16, 16, 30, 16, 16, 16, y)
	case 'G':
		return packGlyph(14, 17, 16, 23, 17, 17, 15, y)
	case 'H':
		return packGlyph(17, 17, 17, 31, 17, 17, 17, y)
	case 'I':
		return packGlyph(14, 4, 4, 4, 4, 4, 14, y)
	case 'J':
		return packGlyph(7, 2, 2, 2, 18, 18, 12, y)
	case 'K':
		return packGlyph(17, 18, 20, 24, 20, 18, 17, y)
	case 'L':
		return packGlyph(16, 16, 16, 16, 16, 16, 31, y)
	case 'M':
		return packGlyph(17, 27, 21, 21, 17, 17, 17, y)
	case 'N':
		return packGlyph(17, 25, 21, 19, 17, 17, 17, y)
	case 'O':
		return packGlyph(14, 17, 17, 17, 17, 17, 14, y)
	case 'P':
		return packGlyph(30, 17, 17, 30, 16, 16, 16, y)
	case 'Q':
		return packGlyph(14, 17, 17, 17, 21, 18, 13, y)
	case 'R':
		return packGlyph(30, 17, 17, 30, 20, 18, 17, y)
	case 'S':
		return packGlyph(15, 16, 16, 14, 1, 1, 30, y)
	case 'T':
		return packGlyph(31, 4, 4, 4, 4, 4, 4, y)
	case 'U':
		return packGlyph(17, 17, 17, 17, 17, 17, 14, y)
	case 'V':
		return packGlyph(17, 17, 17, 17, 17, 10, 4, y)
	case 'W':
		return packGlyph(17, 17, 17, 21, 21, 21, 10, y)
	case 'X':
		return packGlyph(17, 17, 10, 4, 10, 17, 17, y)
	case 'Y':
		return packGlyph(17, 17, 10, 4, 4, 4, 4, y)
	case 'Z':
		return packGlyph(31, 1, 2, 4, 8, 16, 31, y)
	case 'a':
		return packGlyph(0, 0, 14, 1, 15, 17, 15, y)
	case 'b':
		return packGlyph(16, 16, 30, 17, 17, 17, 30, y)
	case 'c':
		return packGlyph(0, 0, 14, 17, 16, 17, 14, y)
	case 'd':
		return packGlyph(1, 1, 15, 17, 17, 17, 15, y)
	case 'e':
		return packGlyph(0, 0, 14, 17, 31, 16, 14, y)
	case 'f':
		return packGlyph(6, 8, 8, 28, 8, 8, 8, y)
	case 'g':
		return packGlyph(0, 0, 15, 17, 15, 1, 14, y)
	case 'h':
		return packGlyph(16, 16, 30, 17, 17, 17, 17, y)
	case 'i':
		return packGlyph(4, 0, 12, 4, 4, 4, 14, y)
	case 'j':
		return packGlyph(2, 0, 6, 2, 2, 18, 12, y)
	case 'k':
		return packGlyph(16, 16, 18, 20, 24, 20, 18, y)
	case 'l':
		return packGlyph(12, 4, 4, 4, 4, 4, 14, y)
	case 'm':
		return packGlyph(0, 0, 26, 21, 21, 21, 21, y)
	case 'n':
		return packGlyph(0, 0, 30, 17, 17, 17, 17, y)
	case 'o':
		return packGlyph(0, 0, 14, 17, 17, 17, 14, y)
	case 'p':
		return packGlyph(0, 0, 30, 17, 30, 16, 16, y)
	case 'q':
		return packGlyph(0, 0, 15, 17, 15, 1, 1, y)
	case 'r':
		return packGlyph(0, 0, 22, 25, 16, 16, 16, y)
	case 's':
		return packGlyph(0, 0, 15, 16, 14, 1, 30, y)
	case 't':
		return packGlyph(8, 8, 28, 8, 8, 9, 6, y)
	case 'u':
		return packGlyph(0, 0, 17, 17, 17, 19, 13, y)
	case 'v':
		return packGlyph(0, 0, 17, 17, 17, 10, 4, y)
	case 'w':
		return packGlyph(0, 0, 17, 17, 21, 21, 10, y)
	case 'x':
		return packGlyph(0, 0, 17, 10, 4, 10, 17, y)
	case 'y':
		return packGlyph(0, 0, 17, 17, 15, 1, 14, y)
	case 'z':
		return packGlyph(0, 0, 31, 2, 4, 8, 31, y)
	case '0':
		return packGlyph(14, 17, 19, 21, 25, 17, 14, y)
	case '1':
		return packGlyph(4, 12, 4, 4, 4, 4, 14, y)
	case '2':
		return packGlyph(14, 17, 1, 2, 4, 8, 31, y)
	case '3':
		return packGlyph(30, 1, 1, 14, 1, 1, 30, y)
	case '4':
		return packGlyph(2, 6, 10, 18, 31, 2, 2, y)
	case '5':
		return packGlyph(31, 16, 16, 30, 1, 1, 30, y)
	case '6':
		return packGlyph(14, 16, 16, 30, 17, 17, 14, y)
	case '7':
		return packGlyph(31, 1, 2, 4, 8, 8, 8, y)
	case '8':
		return packGlyph(14, 17, 17, 14, 17, 17, 14, y)
	case '9':
		return packGlyph(14, 17, 17, 15, 1, 1, 14, y)
	case ' ':
		return 0
	case '.':
		return packGlyph(0, 0, 0, 0, 0, 6, 6, y)
	case ',':
		return packGlyph(0, 0, 0, 0, 6, 6, 4, y)
	case ':':
		return packGlyph(0, 6, 6, 0, 6, 6, 0, y)
	case ';':
		return packGlyph(0, 6, 6, 0, 6, 6, 4, y)
	case '!':
		return packGlyph(4, 4, 4, 4, 4, 0, 4, y)
	case '?':
		return packGlyph(14, 17, 1, 2, 4, 0, 4, y)
	case '-':
		return packGlyph(0, 0, 0, 31, 0, 0, 0, y)
	case '_':
		return packGlyph(0, 0, 0, 0, 0, 0, 31, y)
	case '+':
		return packGlyph(0, 4, 4, 31, 4, 4, 0, y)
	case '/':
		return packGlyph(1, 2, 2, 4, 8, 8, 16, y)
	case '\\':
		return packGlyph(16, 8, 8, 4, 2, 2, 1, y)
	case '(':
		return packGlyph(2, 4, 8, 8, 8, 4, 2, y)
	case ')':
		return packGlyph(8, 4, 2, 2, 2, 4, 8, y)
	case '[':
		return packGlyph(14, 8, 8, 8, 8, 8, 14, y)
	case ']':
		return packGlyph(14, 2, 2, 2, 2, 2, 14, y)
	case '=':
		return packGlyph(0, 0, 31, 0, 31, 0, 0, y)
	}
	return packGlyph(31, 17, 5, 4, 4, 0, 4, y)
}

func glyphRow(r, y int) byte {
	return glyphBits(r, y)
}
