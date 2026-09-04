package graphics

import "unsafe"

type FontMetrics struct {
	Ascent  Scalar
	Descent Scalar
	LineGap Scalar
}

type Font struct {
	Scale   int
	Metrics FontMetrics

	trueType    *trueTypeInfo
	raster      bool
	rasterFirst int
	rasterDense bool
	pixelHeight Scalar
	glyphs      []fontGlyph
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

// RasterGlyph is one pre-rasterized, reusable A8 glyph. It allows constrained
// targets to use antialiased fonts without parsing or rasterizing outlines at
// runtime.
type RasterGlyph struct {
	Codepoint                         int
	Mask                              *Image
	MaskPixels                        []byte
	MaskSource                        *string
	MaskOffset                        int
	MaskWidth, MaskHeight, MaskStride int
	XOffset, YOffset, Advance         Scalar
}

type fontGlyph struct {
	codepoint   int
	rasterScale int
	index       int
	mask        *Image
	maskPixels  []byte
	maskSource  *string
	maskOffset  int
	maskWidth   int
	maskHeight  int
	maskStride  int
	xOffset     Scalar
	yOffset     Scalar
	advance     Scalar
}

func NewBuiltinFont(scale int) *Font {
	if scale < 1 {
		scale = 1
	}
	return &Font{Scale: scale, Metrics: FontMetrics{Ascent: Scalar(7 * scale), Descent: Scalar(2 * scale), LineGap: Scalar(scale)}}
}

// NewRasterFont creates a font from pre-rasterized glyph masks. The masks are
// retained by reference and the small glyph metadata table is copied.
func NewRasterFont(metrics FontMetrics, glyphs []RasterGlyph) *Font {
	font := NewRasterFontCapacity(metrics, len(glyphs))
	for i := 0; i < len(glyphs); i++ {
		font.AddRasterGlyph(glyphs[i])
	}
	return font
}

// NewRasterFontCapacity creates an empty pre-rasterized font and reserves its
// glyph metadata. AddRasterGlyph can then populate it without a temporary
// RasterGlyph slice, which matters for monotonic small-target allocators.
func NewRasterFontCapacity(metrics FontMetrics, capacity int) *Font {
	if capacity < 0 {
		capacity = 0
	}
	return &Font{Metrics: metrics, raster: true, glyphs: make([]fontGlyph, 0, capacity)}
}

// AddRasterGlyph appends one glyph to a font made by NewRasterFontCapacity.
// Glyphs should be added in ascending codepoint order.
func (font *Font) AddRasterGlyph(glyph RasterGlyph) {
	if font == nil || !font.raster {
		return
	}
	index := len(font.glyphs)
	if index == 0 {
		font.rasterFirst = glyph.Codepoint
		font.rasterDense = true
	} else if glyph.Codepoint != font.rasterFirst+index {
		font.rasterDense = false
	}
	stride := glyph.MaskStride
	if stride == 0 {
		stride = glyph.MaskWidth
	}
	font.glyphs = append(font.glyphs, fontGlyph{
		codepoint: glyph.Codepoint, rasterScale: 1, index: glyph.Codepoint,
		mask: glyph.Mask, maskPixels: glyph.MaskPixels,
		maskSource: glyph.MaskSource, maskOffset: glyph.MaskOffset,
		maskWidth: glyph.MaskWidth, maskHeight: glyph.MaskHeight, maskStride: stride,
		xOffset: glyph.XOffset, yOffset: glyph.YOffset, advance: glyph.Advance,
	})
}

func (font *Font) usesRasterGlyphs() bool {
	return font != nil && (font.trueType != nil || font.raster)
}

// NewTrueTypeFont parses a TrueType font and prepares it for antialiased
// rendering at pixelHeight logical pixels. The font data is copied, so callers
// may reuse or release their input buffer after this function returns.
func NewTrueTypeFont(data []byte, pixelHeight Scalar) (*Font, *Error) {
	if pixelHeight <= 0 {
		return nil, &Error{Message: "graphics: TrueType pixel height must be positive"}
	}
	owned := make([]byte, len(data))
	copy(owned, data)
	info, err := initTrueType(owned, 0)
	if err != nil {
		return nil, err
	}
	ascent, descent, lineGap := info.GetFontVMetrics()
	units := ascent - descent
	if units <= 0 {
		return nil, &Error{Message: "graphics: invalid TrueType vertical metrics"}
	}
	ascentMetric := Scalar(ascent) * pixelHeight / Scalar(units)
	descentMetric := Scalar(-descent) * pixelHeight / Scalar(units)
	lineHeight := pixelHeight + Scalar(lineGap)*pixelHeight/Scalar(units)
	gap := lineHeight - ascentMetric - descentMetric
	if gap < 0 {
		gap = 0
	}
	return &Font{
		Metrics: FontMetrics{
			Ascent:  ascentMetric,
			Descent: descentMetric,
			LineGap: gap,
		},
		trueType:    info,
		pixelHeight: pixelHeight,
	}, nil
}

func (font *Font) trueTypeScale() Scalar {
	if font == nil || font.trueType == nil {
		return 0
	}
	ascent, descent, _ := font.trueType.GetFontVMetrics()
	units := ascent - descent
	if units <= 0 {
		return 0
	}
	return font.pixelHeight * ttScaleUnit / Scalar(units)
}

func (font *Font) cachedGlyph(codepoint int) *fontGlyph {
	return font.cachedGlyphAtScale(codepoint, 1)
}

func (font *Font) cachedGlyphAtScale(codepoint, rasterScale int) *fontGlyph {
	if rasterScale < 1 {
		rasterScale = 1
	}
	if rasterScale == 1 && font.rasterDense {
		index := codepoint - font.rasterFirst
		if index >= 0 && index < len(font.glyphs) {
			return &font.glyphs[index]
		}
	}
	for i := 0; i < len(font.glyphs); i++ {
		if font.glyphs[i].codepoint == codepoint && font.glyphs[i].rasterScale == rasterScale {
			return &font.glyphs[i]
		}
	}
	if font.trueType == nil {
		// Pre-rasterized fonts cannot synthesize a new scale or codepoint. Use
		// their question-mark glyph when available.
		if rasterScale == 1 && font.rasterDense {
			index := int('?') - font.rasterFirst
			if index >= 0 && index < len(font.glyphs) {
				return &font.glyphs[index]
			}
		}
		for i := 0; i < len(font.glyphs); i++ {
			if font.glyphs[i].codepoint == '?' && font.glyphs[i].rasterScale == rasterScale {
				return &font.glyphs[i]
			}
		}
		return nil
	}
	info := font.trueType
	scale := font.trueTypeScale() * Scalar(rasterScale)
	index := info.FindGlyphIndex(codepoint)
	advance, _ := info.GetGlyphHMetrics(index)
	x0, y0, x1, y1 := info.GetGlyphBitmapBox(index, scale, scale)
	width, height := x1-x0, y1-y0
	var mask *Image
	if width > 0 && height > 0 {
		pixels := make([]byte, width*height)
		info.MakeGlyphBitmap(pixels, width, height, width, scale, scale, index)
		mask = NewMask(width, height, pixels)
	}
	font.glyphs = append(font.glyphs, fontGlyph{
		codepoint:   codepoint,
		rasterScale: rasterScale,
		index:       index,
		mask:        mask,
		xOffset:     Scalar(x0),
		yOffset:     Scalar(y0),
		advance:     Scalar(advance) * font.trueTypeScale() / ttScaleUnit,
	})
	return &font.glyphs[len(font.glyphs)-1]
}

func (font *Font) kern(left, right int) Scalar {
	if font == nil || font.trueType == nil || left < 0 || right < 0 {
		return 0
	}
	return Scalar(font.trueType.GetGlyphKernAdvance(left, right)) * font.trueTypeScale() / ttScaleUnit
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

func nextUTF8Bytes(text []byte, at int) (int, int) {
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
	lineHeight := font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap
	x, width, height := Scalar(0), Scalar(0), lineHeight
	previous := -1
	for at := 0; at < len(text); {
		r, size := nextUTF8(text, at)
		at += size
		if r == 10 {
			if x > width {
				width = x
			}
			x = 0
			height += lineHeight
			previous = -1
		} else if r == 9 {
			if font.usesRasterGlyphs() {
				space := font.cachedGlyph(' ')
				if space != nil {
					x += space.advance * 4
				}
			} else {
				x += Scalar(6*font.Scale) * 4
			}
			previous = -1
		} else {
			if font.usesRasterGlyphs() {
				glyph := font.cachedGlyph(r)
				if glyph != nil {
					x += font.kern(previous, glyph.index) + glyph.advance
					previous = glyph.index
				}
			} else {
				x += Scalar(6 * font.Scale)
			}
		}
	}
	if x > width {
		width = x
	}
	return TextMetrics{Width: width, Height: height}
}

// MeasureTextBytes is the allocation-free byte-slice counterpart to
// MeasureText. It is useful for rendering mutable fixed-capacity text buffers.
func MeasureTextBytes(font *Font, text []byte) TextMetrics {
	if font == nil {
		return TextMetrics{}
	}
	lineHeight := font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap
	x, width, height := Scalar(0), Scalar(0), lineHeight
	previous := -1
	for at := 0; at < len(text); {
		r, size := nextUTF8Bytes(text, at)
		at += size
		if r == 10 {
			if x > width {
				width = x
			}
			x = 0
			height += lineHeight
			previous = -1
		} else if r == 9 {
			if font.usesRasterGlyphs() {
				space := font.cachedGlyph(' ')
				if space != nil {
					x += space.advance * 4
				}
			} else {
				x += Scalar(6*font.Scale) * 4
			}
			previous = -1
		} else {
			if font.usesRasterGlyphs() {
				glyph := font.cachedGlyph(r)
				if glyph != nil {
					x += font.kern(previous, glyph.index) + glyph.advance
					previous = glyph.index
				}
			} else {
				x += Scalar(6 * font.Scale)
			}
		}
	}
	if x > width {
		width = x
	}
	return TextMetrics{Width: width, Height: height}
}

func glyphRows(r int) [7]byte {
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
	case 'a':
		return [7]byte{0, 0, 14, 1, 15, 17, 15}
	case 'b':
		return [7]byte{16, 16, 30, 17, 17, 17, 30}
	case 'c':
		return [7]byte{0, 0, 14, 17, 16, 17, 14}
	case 'd':
		return [7]byte{1, 1, 15, 17, 17, 17, 15}
	case 'e':
		return [7]byte{0, 0, 14, 17, 31, 16, 14}
	case 'f':
		return [7]byte{6, 8, 8, 28, 8, 8, 8}
	case 'g':
		return [7]byte{0, 0, 15, 17, 15, 1, 14}
	case 'h':
		return [7]byte{16, 16, 30, 17, 17, 17, 17}
	case 'i':
		return [7]byte{4, 0, 12, 4, 4, 4, 14}
	case 'j':
		return [7]byte{2, 0, 6, 2, 2, 18, 12}
	case 'k':
		return [7]byte{16, 16, 18, 20, 24, 20, 18}
	case 'l':
		return [7]byte{12, 4, 4, 4, 4, 4, 14}
	case 'm':
		return [7]byte{0, 0, 26, 21, 21, 21, 21}
	case 'n':
		return [7]byte{0, 0, 30, 17, 17, 17, 17}
	case 'o':
		return [7]byte{0, 0, 14, 17, 17, 17, 14}
	case 'p':
		return [7]byte{0, 0, 30, 17, 30, 16, 16}
	case 'q':
		return [7]byte{0, 0, 15, 17, 15, 1, 1}
	case 'r':
		return [7]byte{0, 0, 22, 25, 16, 16, 16}
	case 's':
		return [7]byte{0, 0, 15, 16, 14, 1, 30}
	case 't':
		return [7]byte{8, 8, 28, 8, 8, 9, 6}
	case 'u':
		return [7]byte{0, 0, 17, 17, 17, 19, 13}
	case 'v':
		return [7]byte{0, 0, 17, 17, 17, 10, 4}
	case 'w':
		return [7]byte{0, 0, 17, 17, 21, 21, 10}
	case 'x':
		return [7]byte{0, 0, 17, 10, 4, 10, 17}
	case 'y':
		return [7]byte{0, 0, 17, 17, 15, 1, 14}
	case 'z':
		return [7]byte{0, 0, 31, 2, 4, 8, 31}
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
		for x := 0; x < 5; {
			for x < 5 && bits&(1<<uint(4-x)) == 0 {
				x++
			}
			start := x
			for x < 5 && bits&(1<<uint(4-x)) != 0 {
				x++
			}
			if start < x {
				s.FillRect(R(
					position.X+Scalar(start)*scale,
					position.Y+Scalar(y)*scale,
					Scalar(x-start)*scale, scale,
				), color)
			}
		}
	}
}

func (s *Surface) drawGlyphMask(mask *Image, x, y int, color Color) {
	if mask == nil || mask.Format != PixelA8 {
		return
	}
	minX, minY := 0, 0
	maxX, maxY := mask.Width, mask.Height
	if x < s.clip.minX {
		minX = s.clip.minX - x
	}
	if y < s.clip.minY {
		minY = s.clip.minY - y
	}
	if x+maxX > s.clip.maxX {
		maxX = s.clip.maxX - x
	}
	if y+maxY > s.clip.maxY {
		maxY = s.clip.maxY - y
	}
	if minX >= maxX || minY >= maxY {
		return
	}
	s.markDirtyRect(pixelRect{minX: x + minX, minY: y + minY, maxX: x + maxX, maxY: y + maxY})
	if s.Format == PixelRGB565 && color.A == 255 && s.blend == BlendSourceOver {
		s.drawGlyphMaskRGB565(mask, x, y, minX, minY, maxX, maxY, color)
		return
	}
	for maskY := minY; maskY < maxY; maskY++ {
		for maskX := minX; maskX < maxX; maskX++ {
			alpha := int(mask.Pixels[maskY*mask.Stride+maskX])
			if alpha != 0 {
				tinted := Color{
					R: byte((int(color.R)*alpha + 127) / 255),
					G: byte((int(color.G)*alpha + 127) / 255),
					B: byte((int(color.B)*alpha + 127) / 255),
					A: byte((int(color.A)*alpha + 127) / 255),
				}
				s.writePixel(x+maskX, y+maskY, tinted)
			}
		}
	}
}

func (s *Surface) drawPositionedGlyph(mask *Image, x, y, xOffset, yOffset Scalar, color Color) {
	if mask == nil {
		return
	}
	if s.transformB != 0 || s.transformC != 0 || s.transformA != s.transformD {
		s.DrawImage(mask, R(0, 0, Scalar(mask.Width), Scalar(mask.Height)),
			R(x+xOffset, y+yOffset, Scalar(mask.Width), Scalar(mask.Height)),
			SamplingNearest, color)
		return
	}
	origin := s.transformPoint(Point{X: x, Y: y})
	drawX := scalarFloor(origin.X + xOffset + 0.5)
	drawY := scalarFloor(origin.Y + yOffset + 0.5)
	s.drawGlyphMask(mask, drawX, drawY, color)
}

func (s *Surface) drawFontGlyph(glyph *fontGlyph, x, y Scalar, color Color) {
	if glyph.mask != nil {
		s.drawPositionedGlyph(glyph.mask, x, y, glyph.xOffset, glyph.yOffset, color)
		return
	}
	if glyph.maskSource != nil && glyph.maskWidth > 0 && glyph.maskHeight > 0 &&
		glyph.maskStride >= glyph.maskWidth && glyph.maskOffset >= 0 &&
		glyph.maskOffset+glyph.maskStride*glyph.maskHeight <= len(*glyph.maskSource) {
		data := *glyph.maskSource
		s.drawPositionedGlyphData(data[glyph.maskOffset:], glyph.maskWidth, glyph.maskHeight,
			glyph.maskStride, x, y, glyph.xOffset, glyph.yOffset, color)
		return
	}
	if glyph.maskWidth <= 0 || glyph.maskHeight <= 0 || glyph.maskStride < glyph.maskWidth ||
		len(glyph.maskPixels) < glyph.maskStride*glyph.maskHeight {
		return
	}
	// Keep cached glyphs compact: a general-purpose Surface is much larger than
	// its immutable A8 dimensions and byte view. Materialize that view on the
	// stack only while drawing the glyph.
	mask := Surface{
		Width: glyph.maskWidth, Height: glyph.maskHeight, Stride: glyph.maskStride,
		Pixels: glyph.maskPixels, Format: PixelA8,
	}
	s.drawPositionedGlyph(&mask, x, y, glyph.xOffset, glyph.yOffset, color)
}

func (s *Surface) drawPositionedGlyphData(data string, width, height, stride int,
	x, y, xOffset, yOffset Scalar, color Color) {
	if s.deviceScale == 1 && s.transformA == 0 && s.transformB == -1 &&
		s.transformC == 1 && s.transformD == 0 {
		// Quarter-turn display surfaces are common on portrait panels whose
		// controller RAM is landscape. Map glyph pixels with integers so exact
		// rotations do not pay for (or depend on) a floating-point inverse affine
		// operation for every covered pixel.
		logicalX := scalarFloor(x + xOffset + 0.5)
		logicalY := scalarFloor(y + yOffset + 0.5)
		translateX := scalarFloor(s.transformTX)
		translateY := scalarFloor(s.transformTY)
		for maskY := 0; maskY < height; maskY++ {
			for maskX := 0; maskX < width; maskX++ {
				alpha := byte(data[maskY*stride+maskX])
				if alpha != 0 {
					s.putPixel(translateX+logicalY+maskY,
						translateY-logicalX-maskX-1, Color{
							R: byte((int(color.R)*int(alpha) + 127) / 255),
							G: byte((int(color.G)*int(alpha) + 127) / 255),
							B: byte((int(color.B)*int(alpha) + 127) / 255),
							A: byte((int(color.A)*int(alpha) + 127) / 255),
						})
				}
			}
		}
		return
	}
	if s.transformB == 0 && s.transformC == 0 && s.transformA == s.transformD {
		origin := s.transformPoint(Point{X: x, Y: y})
		drawX := scalarFloor(origin.X + xOffset + 0.5)
		drawY := scalarFloor(origin.Y + yOffset + 0.5)
		for maskY := 0; maskY < height; maskY++ {
			for maskX := 0; maskX < width; maskX++ {
				alpha := byte(data[maskY*stride+maskX])
				if alpha != 0 {
					s.putPixel(drawX+maskX, drawY+maskY, Color{
						R: byte((int(color.R)*int(alpha) + 127) / 255),
						G: byte((int(color.G)*int(alpha) + 127) / 255),
						B: byte((int(color.B)*int(alpha) + 127) / 255),
						A: byte((int(color.A)*int(alpha) + 127) / 255),
					})
				}
			}
		}
		return
	}
	destination := R(x+xOffset, y+yOffset, Scalar(width), Scalar(height))
	a := Point{X: destination.MinX, Y: destination.MinY}
	b := Point{X: destination.MaxX, Y: destination.MinY}
	c := Point{X: destination.MaxX, Y: destination.MaxY}
	d := Point{X: destination.MinX, Y: destination.MaxY}
	s.transformPointInPlace(&a)
	s.transformPointInPlace(&b)
	s.transformPointInPlace(&c)
	s.transformPointInPlace(&d)
	minX, maxX, minY, maxY := pointBounds4(a, b, c, d)
	determinant := s.transformA*s.transformD - s.transformB*s.transformC
	if determinant == 0 {
		return
	}
	for outputY := scalarFloor(minY); outputY < scalarCeil(maxY); outputY++ {
		for outputX := scalarFloor(minX); outputX < scalarCeil(maxX); outputX++ {
			sx := (Scalar(outputX)+0.5)/s.deviceScale - s.transformTX
			sy := (Scalar(outputY)+0.5)/s.deviceScale - s.transformTY
			localX := (s.transformD*sx - s.transformC*sy) / determinant
			localY := (-s.transformB*sx + s.transformA*sy) / determinant
			u := (localX - destination.MinX) / destination.Width()
			v := (localY - destination.MinY) / destination.Height()
			if u < 0 || u >= 1 || v < 0 || v >= 1 {
				continue
			}
			maskX, maskY := scalarFloor(u*Scalar(width)), scalarFloor(v*Scalar(height))
			alpha := byte(data[maskY*stride+maskX])
			if alpha != 0 {
				s.putPixel(outputX, outputY, Color{
					R: byte((int(color.R)*int(alpha) + 127) / 255),
					G: byte((int(color.G)*int(alpha) + 127) / 255),
					B: byte((int(color.B)*int(alpha) + 127) / 255),
					A: byte((int(color.A)*int(alpha) + 127) / 255),
				})
			}
		}
	}
}

func blendRGB565Channel(source, destination, alpha int) int {
	return (source*alpha + destination*(255-alpha) + 127) / 255
}

func (s *Surface) drawGlyphMaskRGB565(mask *Image, x, y, minX, minY, maxX, maxY int, color Color) {
	solid := encodeRGB565(color)
	for maskY := minY; maskY < maxY; maskY++ {
		for maskX := minX; maskX < maxX; {
			alpha := int(mask.Pixels[maskY*mask.Stride+maskX])
			if alpha == 0 {
				maskX++
				continue
			}
			if alpha == 255 {
				start := maskX
				for maskX < maxX && mask.Pixels[maskY*mask.Stride+maskX] == 255 {
					maskX++
				}
				s.fillRGB565Run(y+maskY, x+start, x+maskX, solid)
				continue
			}
			offset := (y+maskY)*s.Stride + (x+maskX)*2
			destination := decodeRGB565(s.Pixels[offset], s.Pixels[offset+1])
			blended := Color{
				R: byte(blendRGB565Channel(int(color.R), int(destination.R), alpha)),
				G: byte(blendRGB565Channel(int(color.G), int(destination.G), alpha)),
				B: byte(blendRGB565Channel(int(color.B), int(destination.B), alpha)),
				A: 255,
			}
			pixel := encodeRGB565(blended)
			s.Pixels[offset], s.Pixels[offset+1] = byte(pixel), byte(pixel>>8)
			maskX++
		}
	}
}

func (s *Surface) fillRGB565Run(y, minX, maxX int, pixel uint16) {
	start := y*s.Stride + minX*2
	end := y*s.Stride + maxX*2
	base := uintptr(unsafe.Pointer(&s.Pixels[0]))
	if start < end && (base+uintptr(start))&3 != 0 {
		s.Pixels[start], s.Pixels[start+1] = byte(pixel), byte(pixel>>8)
		start += 2
	}
	word := uint32(pixel) | uint32(pixel)<<16
	for start+4 <= end {
		storeSurface32(base+uintptr(start), word)
		start += 4
	}
	if start < end {
		s.Pixels[start], s.Pixels[start+1] = byte(pixel), byte(pixel>>8)
	}
}

func (s *Surface) DrawText(font *Font, baseline Point, text string, color Color) {
	if font == nil {
		return
	}
	lineHeight := font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap
	originX := baseline.X
	x, y := baseline.X, baseline.Y
	previous := -1
	rasterScale := 1
	textScale := s.transformA * s.deviceScale
	if s.transformB == 0.0 && s.transformC == 0.0 && s.transformA == s.transformD && textScale >= 1.0 && textScale <= 4.0 && Scalar(int(textScale)) == textScale {
		rasterScale = int(textScale)
	}
	for at := 0; at < len(text); {
		r, size := nextUTF8(text, at)
		at += size
		if r == 10 {
			x = originX
			y += lineHeight
			previous = -1
			continue
		}
		if r == 9 {
			if font.usesRasterGlyphs() {
				space := font.cachedGlyph(' ')
				if space != nil {
					x += space.advance * 4
				}
			} else {
				x += Scalar(6*font.Scale) * 4
			}
			previous = -1
			continue
		}
		if font.usesRasterGlyphs() {
			glyph := font.cachedGlyphAtScale(r, rasterScale)
			if glyph == nil {
				continue
			}
			x += font.kern(previous, glyph.index)
			if glyph.mask != nil || len(glyph.maskPixels) != 0 || glyph.maskSource != nil {
				// Glyph bitmaps are rasterized without a subpixel shift, so place
				// them on the corresponding nearest pixel and composite the A8 mask
				// directly. A 1:1 glyph upload does not require affine resampling.
				s.drawFontGlyph(glyph, x, y, color)
			}
			x += glyph.advance
			previous = glyph.index
		} else {
			s.drawBuiltinGlyph(font, Point{X: x, Y: y - font.Metrics.Ascent}, r, color)
			x += Scalar(6 * font.Scale)
		}
	}
}

// DrawTextBytes draws UTF-8 from a byte slice without converting it to an
// immutable string.
func (s *Surface) DrawTextBytes(font *Font, baseline Point, text []byte, color Color) {
	if font == nil {
		return
	}
	lineHeight := font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap
	originX := baseline.X
	x, y := baseline.X, baseline.Y
	previous := -1
	rasterScale := 1
	textScale := s.transformA * s.deviceScale
	if s.transformB == 0.0 && s.transformC == 0.0 && s.transformA == s.transformD && textScale >= 1.0 && textScale <= 4.0 && Scalar(int(textScale)) == textScale {
		rasterScale = int(textScale)
	}
	for at := 0; at < len(text); {
		r, size := nextUTF8Bytes(text, at)
		at += size
		if r == 10 {
			x = originX
			y += lineHeight
			previous = -1
			continue
		}
		if r == 9 {
			if font.usesRasterGlyphs() {
				space := font.cachedGlyph(' ')
				if space != nil {
					x += space.advance * 4
				}
			} else {
				x += Scalar(6*font.Scale) * 4
			}
			previous = -1
			continue
		}
		if font.usesRasterGlyphs() {
			glyph := font.cachedGlyphAtScale(r, rasterScale)
			if glyph == nil {
				continue
			}
			x += font.kern(previous, glyph.index)
			if glyph.mask != nil || len(glyph.maskPixels) != 0 || glyph.maskSource != nil {
				s.drawFontGlyph(glyph, x, y, color)
			}
			x += glyph.advance
			previous = glyph.index
		} else {
			s.drawBuiltinGlyph(font, Point{X: x, Y: y - font.Metrics.Ascent}, r, color)
			x += Scalar(6 * font.Scale)
		}
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
