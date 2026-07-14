// TrueType outline parsing and antialiased scan conversion are adapted from
// github.com/tinyrange/gowin/third_party/truetype, itself ported from
// github.com/gonutz/fontstash.go and Fontstash by Mikko Mononen.
//
// This is an altered source version distributed under the zlib license. See
// THIRD_PARTY_NOTICES.md for the complete notice.
package graphics

type trueTypeInfo struct {
	data             []byte
	fontStart        int
	loca             int
	head             int
	glyf             int
	hhea             int
	hmtx             int
	kern             int
	numGlyphs        int
	indexMap         int
	indexToLocFormat int
}

func initTrueType(data []byte, offset int) (*trueTypeInfo, *Error) {
	if offset < 0 || len(data)-offset < 12 {
		return nil, &Error{Message: "graphics: TrueType data is too short"}
	}
	font := &trueTypeInfo{data: data, fontStart: offset}
	cmap := findTable(data, offset, "cmap")
	font.loca = findTable(data, offset, "loca")
	font.head = findTable(data, offset, "head")
	font.glyf = findTable(data, offset, "glyf")
	font.hhea = findTable(data, offset, "hhea")
	font.hmtx = findTable(data, offset, "hmtx")
	font.kern = findTable(data, offset, "kern")
	if cmap == 0 || font.loca == 0 || font.head == 0 || font.glyf == 0 || font.hhea == 0 || font.hmtx == 0 {
		return nil, &Error{Message: "graphics: required TrueType table not found"}
	}
	maxp := findTable(data, offset, "maxp")
	if maxp != 0 {
		font.numGlyphs = int(u16(data, maxp+4))
	} else {
		font.numGlyphs = 4095
	}
	numTables := int(u16(data, cmap+2))
	for i := 0; i < numTables; i++ {
		record := cmap + 4 + 8*i
		platform := int(u16(data, record))
		encoding := int(u16(data, record+2))
		if platform == PLATFORM_ID_MICROSOFT && (encoding == MS_EID_UNICODE_FULL || encoding == MS_EID_UNICODE_BMP) {
			candidate := cmap + int(u32(data, record+4))
			if font.indexMap == 0 || int(u16(data, candidate)) == 12 {
				font.indexMap = candidate
			}
		}
		if platform == PLATFORM_ID_UNICODE && font.indexMap == 0 {
			font.indexMap = cmap + int(u32(data, record+4))
		}
	}
	if font.indexMap == 0 {
		return nil, &Error{Message: "graphics: unsupported TrueType character map"}
	}
	font.indexToLocFormat = int(u16(data, font.head+50))
	if font.indexToLocFormat < 0 || font.indexToLocFormat > 1 {
		return nil, &Error{Message: "graphics: unsupported TrueType glyph location format"}
	}
	return font, nil
}

// This library processes TrueType files: parsing tables, extracting outlines
// and metrics, and rendering antialiased one-channel glyph masks.

const (
	PLATFORM_ID_UNICODE int = iota
	PLATFORM_ID_MAC
	PLATFORM_ID_ISO
	PLATFORM_ID_MICROSOFT
)

const (
	MS_EID_SYMBOL       int = 0
	MS_EID_UNICODE_BMP      = 1
	MS_EID_SHIFTJIS         = 2
	MS_EID_UNICODE_FULL     = 10
)

const (
	vmove uint8 = iota + 1
	vline
	vcurve
)

const (
	ttScaleUnit = 1024.0
)

type ttVertex struct {
	X       int
	Y       int
	CX      int
	CY      int
	Type    uint8
	Padding byte
}

func setTTVertex(vertices []ttVertex, index int, kind uint8, x, y, cx, cy int) {
	vertices[index].Type = kind
	vertices[index].X = x
	vertices[index].Y = y
	vertices[index].CX = cx
	vertices[index].CY = cy
}

func ttFloor(value float64) float64 {
	i := int(value)
	if value < 0 && float64(i) != value {
		i--
	}
	return float64(i)
}

func ttCeil(value float64) float64 {
	i := int(value)
	if value > 0 && float64(i) != value {
		i++
	}
	return float64(i)
}

func ttSqrt(value float64) float64 {
	if value <= 0 {
		return 0
	}
	guess := value
	if guess < 1 {
		guess = 1
	}
	for i := 0; i < 16; i++ {
		guess = (guess + value/guess) * 0.5
	}
	return guess
}

func ttSignedByte(value byte) int {
	result := int(value)
	if result >= 128 {
		result -= 256
	}
	return result
}

func (font *trueTypeInfo) ScaleForPixelHeight(height float64) float64 {
	ascent, descent, _ := font.GetFontVMetrics()
	fheight := float64(ascent - descent)
	if fheight <= 0 {
		return 0
	}
	return height * ttScaleUnit / fheight
}

func (font *trueTypeInfo) GetGlyphBitmapBox(glyph int, scaleX, scaleY float64) (int, int, int, int) {
	return font.GetGlyphBitmapBoxSubpixel(glyph, scaleX, scaleY, 0, 0)
}

func (font *trueTypeInfo) GetCodepointHMetrics(codepoint int) (int, int) {
	return font.GetGlyphHMetrics(font.FindGlyphIndex(codepoint))
}

func (font *trueTypeInfo) GetFontVMetrics() (int, int, int) {
	return int(int16(u16(font.data, font.hhea+4))), int(int16(u16(font.data, font.hhea+6))), int(int16(u16(font.data, font.hhea+8)))
}

func (font *trueTypeInfo) GetGlyphHMetrics(glyphIndex int) (int, int) {
	numOfLongHorMetrics := int(u16(font.data, font.hhea+34))
	if glyphIndex < numOfLongHorMetrics {
		return int(u16(font.data, font.hmtx+4*glyphIndex)), int(int16(u16(font.data, font.hmtx+4*glyphIndex+2)))
	}
	return int(u16(font.data, font.hmtx+4*(numOfLongHorMetrics-1))), int(int16(u16(font.data, font.hmtx+4*numOfLongHorMetrics+2*(glyphIndex-numOfLongHorMetrics))))
}

func (font *trueTypeInfo) GetFontBoundingBox() (int, int, int, int) {
	return int(int16(u16(font.data, font.head+36))),
		int(int16(u16(font.data, font.head+38))),
		int(int16(u16(font.data, font.head+40))),
		int(int16(u16(font.data, font.head+42)))
}

func (font *trueTypeInfo) GetCodepointBitmapBox(codepoint int, scaleX, scaleY float64) (int, int, int, int) {
	return font.GetCodepointBitmapBoxSubpixel(codepoint, scaleX, scaleY, 0, 0)
}

func (font *trueTypeInfo) GetCodepointBitmapBoxSubpixel(codepoint int, scaleX, scaleY, shiftX, shiftY float64) (int, int, int, int) {
	return font.GetGlyphBitmapBoxSubpixel(font.FindGlyphIndex(codepoint), scaleX, scaleY, shiftX, shiftY)
}

func (font *trueTypeInfo) GetCodepointBitmap(scaleX, scaleY float64, codePoint, xoff, yoff int) ([]byte, int, int) {
	return font.GetCodepointBitmapSubpixel(scaleX, scaleY, 0., 0., codePoint, xoff, yoff)
}

func (font *trueTypeInfo) GetCodepointBitmapSubpixel(scaleX, scaleY, shiftX, shiftY float64, codePoint, xoff, yoff int) ([]byte, int, int) {
	return font.GetGlyphBitmapSubpixel(scaleX, scaleY, shiftX, shiftY, font.FindGlyphIndex(codePoint), xoff, yoff)
}

type ttBitmap struct {
	W      int
	H      int
	Stride int
	Pixels []byte
}

func (font *trueTypeInfo) GetGlyphBitmapSubpixel(scaleX, scaleY, shiftX, shiftY float64, glyph, xoff, yoff int) ([]byte, int, int) {
	var gbm ttBitmap
	var width int
	var height int
	vertices := font.GetGlyphShape(glyph)
	if scaleX == 0 {
		scaleX = scaleY
	}
	if scaleY == 0 {
		if scaleX == 0 {
			return nil, 0, 0
		}
		scaleY = scaleX
	}

	ix0, iy0, ix1, iy1 := font.GetGlyphBitmapBoxSubpixel(glyph, scaleX, scaleY, shiftX, shiftY)

	// now we get the size
	gbm.W = ix1 - ix0
	gbm.H = iy1 - iy0
	gbm.Pixels = nil

	width = gbm.W
	height = gbm.H
	xoff = ix0
	yoff = iy0

	if gbm.W != 0 && gbm.H != 0 {
		gbm.Pixels = make([]byte, gbm.W*gbm.H)
		gbm.Stride = gbm.W

		Rasterize(&gbm, 0.35, vertices, scaleX, scaleY, shiftX, shiftY, ix0, iy0, true)
	}

	return gbm.Pixels, width, height
}

type ttPoint struct {
	x float64
	y float64
}

func Rasterize(result *ttBitmap, flatnessInPixels float64, vertices []ttVertex, scaleX, scaleY, shiftX, shiftY float64, xOff, yOff int, invert bool) {
	var scale float64
	if scaleX > scaleY {
		scale = scaleY
	} else {
		scale = scaleX
	}
	windings, windingLengths, windingCount := FlattenCurves(vertices, flatnessInPixels*ttScaleUnit/scale)
	if windings != nil {
		ttRasterizeSupersampled(result, windings, windingLengths, windingCount, scaleX, scaleY, shiftX, shiftY, xOff, yOff, invert)
	}
}

type ttSubPoint struct {
	x int
	y int
}

func ttRasterizeSupersampled(result *ttBitmap, points []ttPoint, contourLengths []int, contourCount int, scaleX, scaleY, shiftX, shiftY float64, offX, offY int, invert bool) {
	scaled := make([]ttSubPoint, len(points))
	yDirection := 1.0
	if invert {
		yDirection = -1.0
	}
	for i := 0; i < len(points); i++ {
		xFloat := points[i].x*scaleX*16.0/ttScaleUnit + shiftX*16.0 - float64(offX*16)
		yFloat := points[i].y*scaleY*yDirection*16.0/ttScaleUnit + shiftY*16.0 - float64(offY*16)
		x := int(xFloat)
		y := int(yFloat)
		scaled[i] = ttSubPoint{x: x, y: y}
	}

	for y := 0; y < result.H; y++ {
		for x := 0; x < result.W; x++ {
			hits := 0
			for sampleY := 0; sampleY < 4; sampleY++ {
				testY := y*16 + 2 + sampleY*4
				for sampleX := 0; sampleX < 4; sampleX++ {
					testX := x*16 + 2 + sampleX*4
					winding := 0
					start := 0
					for contour := 0; contour < contourCount; contour++ {
						end := start + contourLengths[contour]
						previous := end - 1
						for current := start; current < end; current++ {
							x0 := scaled[previous].x
							y0 := scaled[previous].y
							x1 := scaled[current].x
							y1 := scaled[current].y
							if (y0 <= testY && y1 > testY) || (y1 <= testY && y0 > testY) {
								cross := (x1-x0)*(testY-y0) - (testX-x0)*(y1-y0)
								if (y1 > y0 && cross > 0) || (y1 < y0 && cross < 0) {
									if y1 > y0 {
										winding++
									} else {
										winding--
									}
								}
							}
							previous = current
						}
						start = end
					}
					if winding != 0 {
						hits++
					}
				}
			}
			result.Pixels[y*result.Stride+x] = byte((hits*255 + 8) / 16)
		}
	}
}

func FlattenCurves(vertices []ttVertex, objspaceFlatness float64) ([]ttPoint, []int, int) {
	var contourLengths []int
	points := []ttPoint{}

	objspaceFlatnessSquared := objspaceFlatness * objspaceFlatness
	n := 0
	start := 0

	for _, vertex := range vertices {
		if vertex.Type == vmove {
			n++
		}
	}
	numContours := n

	if n == 0 {
		return nil, nil, 0
	}

	contourLengths = make([]int, n)

	var x float64
	var y float64
	n = -1
	for _, vertex := range vertices {
		switch vertex.Type {
		case vmove:
			if n >= 0 {
				contourLengths[n] = len(points) - start
			}
			n++
			start = len(points)

			x = float64(vertex.X)
			y = float64(vertex.Y)
			points = append(points, ttPoint{x, y})
		case vline:
			x = float64(vertex.X)
			y = float64(vertex.Y)
			points = append(points, ttPoint{x, y})
		case vcurve:
			tesselateCurve(&points, x, y, float64(vertex.CX), float64(vertex.CY), float64(vertex.X), float64(vertex.Y), objspaceFlatnessSquared, 0)
			x = float64(vertex.X)
			y = float64(vertex.Y)
		}
		contourLengths[n] = len(points) - start
	}
	return points, contourLengths, numContours
}

// tesselate until threshold p is happy... @TODO warped to compensate for non-linear stretching
func tesselateCurve(points *[]ttPoint, x0, y0, x1, y1, x2, y2, objspaceFlatnessSquared float64, n int) int {
	// midpoint
	mx := (x0 + 2*x1 + x2) / 4
	my := (y0 + 2*y1 + y2) / 4
	// versus directly drawn line
	dx := (x0+x2)/2 - mx
	dy := (y0+y2)/2 - my
	if n > 16 {
		return 1
	}
	if dx*dx+dy*dy > objspaceFlatnessSquared { // half-pixel error allowed... need to be smaller if AA
		tesselateCurve(points, x0, y0, (x0+x1)/2, (y0+y1)/2, mx, my, objspaceFlatnessSquared, n+1)
		tesselateCurve(points, mx, my, (x1+x2)/2, (y1+y2)/2, x2, y2, objspaceFlatnessSquared, n+1)
	} else {
		*points = append(*points, ttPoint{x2, y2})
	}
	return 1
}

func (font *trueTypeInfo) GetGlyphBitmapBoxSubpixel(glyph int, scaleX, scaleY, shiftX, shiftY float64) (int, int, int, int) {
	result, x0, y0, x1, y1 := font.GetGlyphBox(glyph)
	if !result {
		x0 = 0
		y0 = 0
		x1 = 0
		y1 = 0
	}
	fx0 := ttFloor(float64(x0)*scaleX/ttScaleUnit + shiftX)
	fy0 := ttCeil(float64(y1)*scaleY/ttScaleUnit + shiftY)
	fx1 := ttCeil(float64(x1)*scaleX/ttScaleUnit + shiftX)
	fy1 := ttFloor(float64(y0)*scaleY/ttScaleUnit + shiftY)
	ix0 := int(fx0)
	iy0 := -int(fy0)
	ix1 := int(fx1)
	iy1 := -int(fy1)
	return ix0, iy0, ix1, iy1
}

func (font *trueTypeInfo) GetGlyphBox(glyph int) (bool, int, int, int, int) {
	g := font.GetGlyphOffset(glyph)
	if g < 0 {
		return false, 0, 0, 0, 0
	}

	x0 := int(int16(u16(font.data, g+2)))
	y0 := int(int16(u16(font.data, g+4)))
	x1 := int(int16(u16(font.data, g+6)))
	y1 := int(int16(u16(font.data, g+8)))
	return true, x0, y0, x1, y1
}

func (font *trueTypeInfo) GetGlyphShape(glyphIndex int) []ttVertex {
	g := font.GetGlyphOffset(glyphIndex)
	if g < 0 {
		return nil
	}
	numberOfContours := int(int16(u16(font.data, g)))
	if numberOfContours > 0 {
		return font.getSimpleGlyphShape(g, numberOfContours)
	}
	if numberOfContours == -1 {
		return font.getCompoundGlyphShape(g)
	}
	return nil
}

func (font *trueTypeInfo) getSimpleGlyphShape(g, numberOfContours int) []ttVertex {
	data := font.data
	endPoints := g + 10
	instructionLength := int(u16(data, g+10+numberOfContours*2))
	pointData := g + 10 + numberOfContours*2 + 2 + instructionLength
	pointCount := 1 + int(u16(data, endPoints+numberOfContours*2-2))

	flags := make([]byte, pointCount)
	repeats := 0
	var flag byte
	for i := 0; i < pointCount; i++ {
		if repeats == 0 {
			flag = data[pointData]
			pointData++
			if flag&8 != 0 {
				repeats = int(data[pointData])
				pointData++
			}
		} else {
			repeats--
		}
		flags[i] = flag
	}

	xs := make([]int, pointCount)
	x := 0
	for i := 0; i < pointCount; i++ {
		flag = flags[i]
		if flag&2 != 0 {
			delta := int(data[pointData])
			pointData++
			if flag&16 != 0 {
				x += delta
			} else {
				x -= delta
			}
		} else if flag&16 == 0 {
			x += int(int16(u16(data, pointData)))
			pointData += 2
		}
		xs[i] = x
	}

	ys := make([]int, pointCount)
	y := 0
	for i := 0; i < pointCount; i++ {
		flag = flags[i]
		if flag&4 != 0 {
			delta := int(data[pointData])
			pointData++
			if flag&32 != 0 {
				y += delta
			} else {
				y -= delta
			}
		} else if flag&32 == 0 {
			y += int(int16(u16(data, pointData)))
			pointData += 2
		}
		ys[i] = y
	}

	vertices := make([]ttVertex, pointCount+2*numberOfContours)
	vertexCount := 0
	contourStart := 0
	for contour := 0; contour < numberOfContours; contour++ {
		contourEnd := int(u16(data, endPoints+contour*2))
		firstOn := flags[contourStart]&1 != 0
		lastOn := flags[contourEnd]&1 != 0
		startX := 0
		startY := 0
		at := contourStart
		if firstOn {
			startX = xs[contourStart]
			startY = ys[contourStart]
			at++
		} else if lastOn {
			startX = xs[contourEnd]
			startY = ys[contourEnd]
		} else {
			startX = (xs[contourStart] + xs[contourEnd]) / 2
			startY = (ys[contourStart] + ys[contourEnd]) / 2
		}
		setTTVertex(vertices, vertexCount, vmove, startX, startY, 0, 0)
		vertexCount++

		controlX := 0
		controlY := 0
		previousOff := false
		for at <= contourEnd {
			pointX := xs[at]
			pointY := ys[at]
			onCurve := flags[at]&1 != 0
			if !onCurve {
				if previousOff {
					midX := (controlX + pointX) / 2
					midY := (controlY + pointY) / 2
					setTTVertex(vertices, vertexCount, vcurve, midX, midY, controlX, controlY)
					vertexCount++
				}
				controlX = pointX
				controlY = pointY
				previousOff = true
			} else {
				if previousOff {
					setTTVertex(vertices, vertexCount, vcurve, pointX, pointY, controlX, controlY)
				} else {
					setTTVertex(vertices, vertexCount, vline, pointX, pointY, 0, 0)
				}
				vertexCount++
				previousOff = false
			}
			at++
		}
		if previousOff {
			setTTVertex(vertices, vertexCount, vcurve, startX, startY, controlX, controlY)
		} else {
			setTTVertex(vertices, vertexCount, vline, startX, startY, 0, 0)
		}
		vertexCount++
		contourStart = contourEnd + 1
	}
	return vertices[:vertexCount]
}

func (font *trueTypeInfo) getCompoundGlyphShape(g int) []ttVertex {
	data := font.data
	component := g + 10
	vertices := make([]ttVertex, 0, 32)
	more := true
	for more {
		flags := int(u16(data, component))
		component += 2
		glyphIndex := int(u16(data, component))
		component += 2

		a := 1.0
		b := 0.0
		c := 0.0
		d := 1.0
		tx := 0.0
		ty := 0.0
		if flags&2 == 0 {
			return nil
		}
		if flags&1 != 0 {
			rawTX := int(int16(u16(data, component)))
			tx = float64(rawTX)
			component += 2
			rawTY := int(int16(u16(data, component)))
			ty = float64(rawTY)
			component += 2
		} else {
			rawTX := ttSignedByte(data[component])
			tx = float64(rawTX)
			component++
			rawTY := ttSignedByte(data[component])
			ty = float64(rawTY)
			component++
		}
		if flags&(1<<3) != 0 {
			rawScale := int(int16(u16(data, component)))
			d = float64(rawScale) / 16384.0
			component += 2
			a = d
		} else if flags&(1<<6) != 0 {
			rawXScale := int(int16(u16(data, component)))
			a = float64(rawXScale) / 16384.0
			component += 2
			rawYScale := int(int16(u16(data, component)))
			d = float64(rawYScale) / 16384.0
			component += 2
		} else if flags&(1<<7) != 0 {
			rawA := int(int16(u16(data, component)))
			a = float64(rawA) / 16384.0
			component += 2
			rawB := int(int16(u16(data, component)))
			b = float64(rawB) / 16384.0
			component += 2
			rawC := int(int16(u16(data, component)))
			c = float64(rawC) / 16384.0
			component += 2
			rawD := int(int16(u16(data, component)))
			d = float64(rawD) / 16384.0
			component += 2
		}

		xScale := ttSqrt(a*a + b*b)
		yScale := ttSqrt(c*c + d*d)
		part := font.GetGlyphShape(glyphIndex)
		for i := 0; i < len(part); i++ {
			oldX := part[i].X
			oldY := part[i].Y
			newXFloat := xScale * (a*float64(oldX) + c*float64(oldY) + tx)
			newYFloat := yScale * (b*float64(oldX) + d*float64(oldY) + ty)
			newX := int(newXFloat)
			newY := int(newYFloat)
			oldCX := part[i].CX
			oldCY := part[i].CY
			newCXFloat := xScale * (a*float64(oldCX) + c*float64(oldCY) + tx)
			newCYFloat := yScale * (b*float64(oldCX) + d*float64(oldCY) + ty)
			newCX := int(newCXFloat)
			newCY := int(newCYFloat)
			setTTVertex(part, i, part[i].Type, newX, newY, newCX, newCY)
		}
		vertices = append(vertices, part...)
		more = flags&(1<<5) != 0
	}
	return vertices
}
func (font *trueTypeInfo) GetGlyphOffset(glyphIndex int) int {
	if glyphIndex >= font.numGlyphs {
		// Glyph index out of range
		return -1
	}
	if font.indexToLocFormat >= 2 {
		// Unknown index-glyph map format
		return -1
	}

	var g1 int
	var g2 int

	if font.indexToLocFormat == 0 {
		offset1 := int(u16(font.data, font.loca+glyphIndex*2)) * 2
		offset2 := int(u16(font.data, font.loca+glyphIndex*2+2)) * 2
		g1 = font.glyf + offset1
		g2 = font.glyf + offset2
	} else {
		offset1 := int(u32(font.data, font.loca+glyphIndex*4))
		offset2 := int(u32(font.data, font.loca+glyphIndex*4+4))
		g1 = font.glyf + offset1
		g2 = font.glyf + offset2
	}

	if g1 == g2 {
		// length is 0
		return -1
	}
	return g1
}

func (font *trueTypeInfo) MakeCodepointBitmap(output []byte, outW, outH, outStride int, scaleX, scaleY float64, codepoint int) []byte {
	return font.MakeCodepointBitmapSubpixel(output, outW, outH, outStride, scaleX, scaleY, 0, 0, codepoint)
}

func (font *trueTypeInfo) MakeCodepointBitmapSubpixel(output []byte, outW, outH, outStride int, scaleX, scaleY, shiftX, shiftY float64, codepoint int) []byte {
	return font.MakeGlyphBitmapSubpixel(output, outW, outH, outStride, scaleX, scaleY, shiftX, shiftY, font.FindGlyphIndex(codepoint))
}

func (font *trueTypeInfo) MakeGlyphBitmap(output []byte, outW, outH, outStride int, scaleX, scaleY float64, glyph int) []byte {
	return font.MakeGlyphBitmapSubpixel(output, outW, outH, outStride, scaleX, scaleY, 0, 0, glyph)
}

func (font *trueTypeInfo) MakeGlyphBitmapSubpixel(output []byte, outW, outH, outStride int, scaleX, scaleY, shiftX, shiftY float64, glyph int) []byte {
	var gbm ttBitmap
	vertices := font.GetGlyphShape(glyph)

	ix0, iy0, _, _ := font.GetGlyphBitmapBoxSubpixel(glyph, scaleX, scaleY, shiftX, shiftY)
	gbm.W = outW
	gbm.H = outH
	gbm.Stride = outStride

	if gbm.W > 0 && gbm.H > 0 {
		gbm.Pixels = output
		Rasterize(&gbm, 0.35, vertices, scaleX, scaleY, shiftX, shiftY, ix0, iy0, true)
	}
	return gbm.Pixels
}

func (font *trueTypeInfo) GetCodepointKernAdvance(ch1, ch2 int) int {
	if font.kern == 0 {
		return 0
	}
	return font.GetGlyphKernAdvance(font.FindGlyphIndex(ch1), font.FindGlyphIndex(ch2))
}

func (font *trueTypeInfo) GetGlyphKernAdvance(glyph1, glyph2 int) int {
	data := font.kern

	// we only look at the first table. it must be 'horizontal' and format 0.
	if font.kern == 0 {
		return 0
	}
	if u16(font.data, data+2) < 1 { // number of tables, need at least 1
		return 0
	}
	if u16(font.data, data+8) != 1 { // horizontal flag must be set in format
		return 0
	}

	l := 0
	r := int(u16(font.data, data+10)) - 1
	needle := uint(glyph1)<<16 | uint(glyph2)
	for l <= r {
		m := (l + r) >> 1
		straw := uint(u32(font.data, data+18+(m*6))) // note: unaligned read
		if needle < straw {
			r = m - 1
		} else if needle > straw {
			l = m + 1
		} else {
			return int(int16(u16(font.data, data+22+(m*6))))
		}
	}
	return 0
}

func (font *trueTypeInfo) FindGlyphIndex(unicodeCodepoint int) int {
	data := font.data
	indexMap := font.indexMap

	format := int(u16(data, indexMap))
	if format == 0 { // apple byte encoding
		numBytes := int(u16(data, indexMap+2))
		if unicodeCodepoint < numBytes-6 {
			return int(data[indexMap+6+unicodeCodepoint])
		}
		return 0
	} else if format == 6 {
		first := int(u16(data, indexMap+6))
		count := int(u16(data, indexMap+8))
		if unicodeCodepoint >= first && unicodeCodepoint < first+count {
			return int(u16(data, indexMap+10+(unicodeCodepoint-first)*2))
		}
		return 0
	} else if format == 2 {
		return 0
	} else if format == 4 {
		segcount := int(u16(data, indexMap+6) >> 1)
		searchRange := int(u16(data, indexMap+8) >> 1)
		entrySelector := int(u16(data, indexMap+10))
		rangeShift := int(u16(data, indexMap+12) >> 1)

		endCount := indexMap + 14
		search := endCount

		if unicodeCodepoint > 0xffff {
			return 0
		}

		if unicodeCodepoint >= int(u16(data, search+rangeShift*2)) {
			search += rangeShift * 2
		}

		search -= 2
		for entrySelector > 0 {
			searchRange = searchRange >> 1
			// start := int(u16(data, search+2+segcount*2+2))
			// end := int(u16(data, search+2))
			// start := int(u16(data, search+searchRange*2+segcount*2+2))
			end := int(u16(data, search+searchRange*2))
			if unicodeCodepoint > end {
				search += searchRange * 2
			}
			entrySelector--
		}
		search += 2

		item := ((search - endCount) >> 1)

		if !(unicodeCodepoint <= int(u16(data, endCount+2*item))) {
			return 0
		}
		start := int(u16(data, indexMap+14+segcount*2+2+2*item))
		// end := int(u16(data, indexMap+14+2+2*item))
		if unicodeCodepoint < start {
			return 0
		}

		offset := int(u16(data, indexMap+14+segcount*6+2+2*item))
		if offset == 0 {
			return unicodeCodepoint + int(int16(u16(data, indexMap+14+segcount*4+2+2*item)))
		}
		return int(u16(data, offset+(unicodeCodepoint-start)*2+indexMap+14+segcount*6+2+2*item))
	} else if format == 12 || format == 13 {
		ngroups := int(u32(data, indexMap+12))
		low := 0
		high := ngroups
		for low < high {
			mid := low + ((high - low) >> 1)
			startChar := int(u32(data, indexMap+16+mid*12))
			endChar := int(u32(data, indexMap+16+mid*12+4))
			if unicodeCodepoint < startChar {
				high = mid
			} else if unicodeCodepoint > endChar {
				low = mid + 1
			} else {
				startGlyph := int(u32(data, indexMap+16+mid*12+8))
				if format == 12 {
					return startGlyph + unicodeCodepoint - startChar
				} else { // format == 13
					return startGlyph
				}
			}
		}
		return 0 // not found
	}
	return 0
}

func findTable(data []byte, offset int, tag string) int {
	numTables := int(u16(data, offset+4))
	tableDir := offset + 12
	for i := 0; i < numTables; i++ {
		loc := tableDir + 16*i
		if string(data[loc:loc+4]) == tag {
			return int(u32(data, loc+8))
		}
	}
	return 0
}

func u32(b []byte, i int) uint32 {
	return uint32(b[i])<<24 | uint32(b[i+1])<<16 | uint32(b[i+2])<<8 | uint32(b[i+3])
}

// u16 returns the big-endian uint16 at b[i:].
func u16(b []byte, i int) uint16 {
	return uint16(b[i])<<8 | uint16(b[i+1])
}

func isFont(data []byte) bool {
	if tag4(data, '1', 0, 0, 0) {
		return true
	}
	if string(data[0:4]) == "typ1" {
		return true
	}
	if string(data[0:4]) == "OTTO" {
		return true
	}
	if tag4(data, 0, 1, 0, 0) {
		return true
	}
	return false
}

func tag4(data []byte, c0, c1, c2, c3 byte) bool {
	return data[0] == c0 && data[1] == c1 && data[2] == c2 && data[3] == c3
}
