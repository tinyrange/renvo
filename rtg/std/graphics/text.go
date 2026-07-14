package graphics

type FontMetrics struct {
	Ascent  Scalar
	Descent Scalar
	LineGap Scalar
}

type Font struct {
	Scale   int
	Metrics FontMetrics
}

type TextMetrics struct {
	Width  Scalar
	Height Scalar
}

type Glyph struct {
	Mask    *Image
	Source  Rect
	X       Scalar
	Y       Scalar
	Advance Scalar
}

func NewBuiltinFont(scale int) *Font {
	if scale < 1 {
		scale = 1
	}
	return &Font{Scale: scale, Metrics: FontMetrics{Ascent: Scalar(7 * scale), Descent: Scalar(2 * scale), LineGap: Scalar(scale)}}
}

func nextUTF8(text string, at int) (int, int) {
	if at >= len(text) {
		return 0, 0
	}
	b0 := text[at]
	if b0 < 0x80 {
		return int(b0), 1
	}
	if b0&0xe0 == 0xc0 && at+1 < len(text) {
		return int(b0&0x1f)<<6 | int(text[at+1]&0x3f), 2
	}
	if b0&0xf0 == 0xe0 && at+2 < len(text) {
		return int(b0&0x0f)<<12 | int(text[at+1]&0x3f)<<6 | int(text[at+2]&0x3f), 3
	}
	if b0&0xf8 == 0xf0 && at+3 < len(text) {
		return int(b0&7)<<18 | int(text[at+1]&0x3f)<<12 | int(text[at+2]&0x3f)<<6 | int(text[at+3]&0x3f), 4
	}
	return 0xfffd, 1
}

func MeasureText(font *Font, text string) TextMetrics {
	if font == nil {
		return TextMetrics{}
	}
	advance := Scalar(6 * font.Scale)
	lineHeight := font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap
	x, width, height := Scalar(0), Scalar(0), lineHeight
	for at := 0; at < len(text); {
		r, size := nextUTF8(text, at)
		at += size
		if r == 10 {
			if x > width {
				width = x
			}
			x = 0
			height += lineHeight
		} else if r == 9 {
			x += advance * 4
		} else {
			x += advance
		}
	}
	if x > width {
		width = x
	}
	return TextMetrics{Width: width, Height: height}
}

func glyphRows(r int) [7]byte {
	if r >= 'a' && r <= 'z' {
		r -= 'a' - 'A'
	}
	switch r {
	case 'A':
		return [7]byte{14, 17, 17, 31, 17, 17, 17}
	case 'B':
		return [7]byte{30, 17, 17, 30, 17, 17, 30}
	case 'C':
		return [7]byte{14, 17, 16, 16, 16, 17, 14}
	case 'D':
		return [7]byte{30, 17, 17, 17, 17, 17, 30}
	case 'E':
		return [7]byte{31, 16, 16, 30, 16, 16, 31}
	case 'F':
		return [7]byte{31, 16, 16, 30, 16, 16, 16}
	case 'G':
		return [7]byte{14, 17, 16, 23, 17, 17, 15}
	case 'H':
		return [7]byte{17, 17, 17, 31, 17, 17, 17}
	case 'I':
		return [7]byte{14, 4, 4, 4, 4, 4, 14}
	case 'J':
		return [7]byte{7, 2, 2, 2, 18, 18, 12}
	case 'K':
		return [7]byte{17, 18, 20, 24, 20, 18, 17}
	case 'L':
		return [7]byte{16, 16, 16, 16, 16, 16, 31}
	case 'M':
		return [7]byte{17, 27, 21, 21, 17, 17, 17}
	case 'N':
		return [7]byte{17, 25, 21, 19, 17, 17, 17}
	case 'O':
		return [7]byte{14, 17, 17, 17, 17, 17, 14}
	case 'P':
		return [7]byte{30, 17, 17, 30, 16, 16, 16}
	case 'Q':
		return [7]byte{14, 17, 17, 17, 21, 18, 13}
	case 'R':
		return [7]byte{30, 17, 17, 30, 20, 18, 17}
	case 'S':
		return [7]byte{15, 16, 16, 14, 1, 1, 30}
	case 'T':
		return [7]byte{31, 4, 4, 4, 4, 4, 4}
	case 'U':
		return [7]byte{17, 17, 17, 17, 17, 17, 14}
	case 'V':
		return [7]byte{17, 17, 17, 17, 17, 10, 4}
	case 'W':
		return [7]byte{17, 17, 17, 21, 21, 21, 10}
	case 'X':
		return [7]byte{17, 17, 10, 4, 10, 17, 17}
	case 'Y':
		return [7]byte{17, 17, 10, 4, 4, 4, 4}
	case 'Z':
		return [7]byte{31, 1, 2, 4, 8, 16, 31}
	case '0':
		return [7]byte{14, 17, 19, 21, 25, 17, 14}
	case '1':
		return [7]byte{4, 12, 4, 4, 4, 4, 14}
	case '2':
		return [7]byte{14, 17, 1, 2, 4, 8, 31}
	case '3':
		return [7]byte{30, 1, 1, 14, 1, 1, 30}
	case '4':
		return [7]byte{2, 6, 10, 18, 31, 2, 2}
	case '5':
		return [7]byte{31, 16, 16, 30, 1, 1, 30}
	case '6':
		return [7]byte{14, 16, 16, 30, 17, 17, 14}
	case '7':
		return [7]byte{31, 1, 2, 4, 8, 8, 8}
	case '8':
		return [7]byte{14, 17, 17, 14, 17, 17, 14}
	case '9':
		return [7]byte{14, 17, 17, 15, 1, 1, 14}
	case '.':
		return [7]byte{0, 0, 0, 0, 0, 6, 6}
	case ',':
		return [7]byte{0, 0, 0, 0, 6, 6, 4}
	case ':':
		return [7]byte{0, 6, 6, 0, 6, 6, 0}
	case ';':
		return [7]byte{0, 6, 6, 0, 6, 6, 4}
	case '!':
		return [7]byte{4, 4, 4, 4, 4, 0, 4}
	case '?':
		return [7]byte{14, 17, 1, 2, 4, 0, 4}
	case '-':
		return [7]byte{0, 0, 0, 31, 0, 0, 0}
	case '_':
		return [7]byte{0, 0, 0, 0, 0, 0, 31}
	case '+':
		return [7]byte{0, 4, 4, 31, 4, 4, 0}
	case '/':
		return [7]byte{1, 2, 2, 4, 8, 8, 16}
	case '\\':
		return [7]byte{16, 8, 8, 4, 2, 2, 1}
	case '(':
		return [7]byte{2, 4, 8, 8, 8, 4, 2}
	case ')':
		return [7]byte{8, 4, 2, 2, 2, 4, 8}
	case '[':
		return [7]byte{14, 8, 8, 8, 8, 8, 14}
	case ']':
		return [7]byte{14, 2, 2, 2, 2, 2, 14}
	case '=':
		return [7]byte{0, 0, 31, 0, 31, 0, 0}
	case ' ':
		return [7]byte{}
	}
	return [7]byte{31, 17, 5, 4, 4, 0, 4}
}

func (s *Surface) drawBuiltinGlyph(font *Font, position Point, r int, color Color) {
	scale := Scalar(font.Scale)
	for y := 0; y < 7; y++ {
		bits := glyphRow(r, y)
		for x := 0; x < 5; x++ {
			if bits&(1<<uint(4-x)) != 0 {
				s.FillRect(R(position.X+Scalar(x)*scale, position.Y+Scalar(y)*scale, scale, scale), color)
			}
		}
	}
}

func (s *Surface) DrawText(font *Font, baseline Point, text string, color Color) {
	if font == nil {
		return
	}
	advance := Scalar(6 * font.Scale)
	lineHeight := font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap
	originX := baseline.X
	x, y := baseline.X, baseline.Y-font.Metrics.Ascent
	for at := 0; at < len(text); {
		r, size := nextUTF8(text, at)
		at += size
		if r == 10 {
			x = originX
			y += lineHeight
			continue
		}
		if r == 9 {
			x += advance * 4
			continue
		}
		s.drawBuiltinGlyph(font, Point{X: x, Y: y}, r, color)
		x += advance
	}
}

func (s *Surface) DrawGlyphRun(origin Point, glyphs []Glyph, color Color) {
	for i := 0; i < len(glyphs); i++ {
		glyph := glyphs[i]
		if glyph.Mask == nil {
			continue
		}
		s.DrawImage(glyph.Mask, glyph.Source, R(origin.X+glyph.X, origin.Y+glyph.Y, glyph.Source.Width(), glyph.Source.Height()), SamplingNearest, color)
	}
}
