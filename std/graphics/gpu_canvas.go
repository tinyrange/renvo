package graphics

type drawCommandKind int

const (
	drawCommandSolid drawCommandKind = iota
	drawCommandRGBA
	drawCommandMask
)

type drawVertex struct {
	x Scalar
	y Scalar
	u Scalar
	v Scalar
}

type drawCommand struct {
	kind     drawCommandKind
	blend    BlendMode
	sampling Sampling
	clip     pixelRect
	first    int
	count    int
	image    int
	color    Color
}

type drawList struct {
	vertices []drawVertex
	commands []drawCommand
	images   []*Image
}

func (l *drawList) reset() {
	l.vertices = l.vertices[:0]
	l.commands = l.commands[:0]
	l.images = l.images[:0]
}

type gpuCanvas struct {
	width       int
	height      int
	deviceScale Scalar
	blend       BlendMode
	clip        pixelRect
	clips       []pixelRect
	transform   Mat2x3
	transforms  []Mat2x3
	damage      []pixelRect
	damageDepth int
	list        drawList
	crossings   []pathCrossing
	ellipses    []ellipseMaskEntry
}

type pathCrossing struct {
	x         Scalar
	direction int
}

type ellipseMaskEntry struct {
	width  int
	height int
	stroke int
	image  *Image
}

const gpuCircleSegments = 16

func gpuCirclePoint(index int) Point {
	index %= gpuCircleSegments
	base := index
	if index > 4 && index <= 8 {
		base = 8 - index
	} else if index > 8 && index <= 12 {
		base = index - 8
	} else if index > 12 {
		base = 16 - index
	}
	point := Point{}
	if base == 0 {
		point = Point{X: 1}
	} else if base == 1 {
		point = Point{X: 0.9238795325, Y: 0.3826834324}
	} else if base == 2 {
		point = Point{X: 0.7071067812, Y: 0.7071067812}
	} else if base == 3 {
		point = Point{X: 0.3826834324, Y: 0.9238795325}
	} else {
		point = Point{Y: 1}
	}
	if index > 4 && index <= 12 {
		point.X = -point.X
	}
	if index > 8 {
		point.Y = -point.Y
	}
	return point
}

func newGPUCanvas() *gpuCanvas { return &gpuCanvas{} }

func (c *gpuCanvas) beginFrame(width, height int, scale Scalar) {
	c.width = width
	c.height = height
	if scale < 1 {
		scale = 1
	}
	c.deviceScale = scale
	c.blend = BlendSourceOver
	c.clip = pixelRect{maxX: width, maxY: height}
	c.clips = c.clips[:0]
	c.transform = Identity()
	c.transforms = c.transforms[:0]
	c.damage = c.damage[:0]
	c.damageDepth = 0
	c.list.reset()
}

func (c *gpuCanvas) endFrame() (*drawList, []pixelRect) {
	return &c.list, c.damage
}

func (c *gpuCanvas) cancelFrame() {
	c.damage = c.damage[:0]
	c.list.reset()
}

func (c *gpuCanvas) transformPoint(point Point) Point {
	return Point{
		X: (c.transform.A*point.X + c.transform.C*point.Y + c.transform.TX) * c.deviceScale,
		Y: (c.transform.B*point.X + c.transform.D*point.Y + c.transform.TY) * c.deviceScale,
	}
}

func (c *gpuCanvas) transformedRect(rect Rect) pixelRect {
	a := c.transformPoint(Point{X: rect.MinX, Y: rect.MinY})
	b := c.transformPoint(Point{X: rect.MaxX, Y: rect.MinY})
	d := c.transformPoint(Point{X: rect.MinX, Y: rect.MaxY})
	e := c.transformPoint(Point{X: rect.MaxX, Y: rect.MaxY})
	minX, maxX, minY, maxY := pointBounds4(a, b, e, d)
	return pixelRect{minX: scalarFloor(minX), minY: scalarFloor(minY), maxX: scalarCeil(maxX), maxY: scalarCeil(maxY)}
}

func (c *gpuCanvas) SetBlendMode(mode BlendMode) { c.blend = mode }

func (c *gpuCanvas) SetTransform(matrix *Mat2x3) {
	if matrix == nil {
		c.ResetTransform()
		return
	}
	c.transform = Mat2x3{
		A: linearArgumentScalar(matrix.A), B: linearArgumentScalar(matrix.B),
		C: linearArgumentScalar(matrix.C), D: linearArgumentScalar(matrix.D),
		TX: matrixScalar(matrix.TX), TY: matrixScalar(matrix.TY),
	}
}

func (c *gpuCanvas) SetAffine(a, b, d, e, tx, ty Scalar) {
	c.transform = Mat2x3{A: linearArgumentScalar(a), B: linearArgumentScalar(b), C: linearArgumentScalar(d), D: linearArgumentScalar(e), TX: tx, TY: ty}
}

func (c *gpuCanvas) SetLinear(a, b, d, e Scalar) {
	c.SetAxisX(a, b)
	c.SetAxisY(d, e)
}

func (c *gpuCanvas) SetAxisX(x, y Scalar) {
	c.transform.A = linearArgumentScalar(x)
	c.transform.B = linearArgumentScalar(y)
}

func (c *gpuCanvas) SetAxisY(x, y Scalar) {
	c.transform.C = linearArgumentScalar(x)
	c.transform.D = linearArgumentScalar(y)
}

func (c *gpuCanvas) SetOffset(tx, ty Scalar) { c.transform.TX, c.transform.TY = tx, ty }

func (c *gpuCanvas) SetTranslation(x, y Scalar) {
	c.transform = Translate(x, y)
}

func (c *gpuCanvas) ResetTransform() { c.transform = Identity() }

func (c *gpuCanvas) PushTransform() { c.transforms = append(c.transforms, c.transform) }

func (c *gpuCanvas) PopTransform() {
	if len(c.transforms) == 0 {
		return
	}
	c.transform = c.transforms[len(c.transforms)-1]
	c.transforms = c.transforms[:len(c.transforms)-1]
}

func (c *gpuCanvas) PushClipRect(rect Rect) {
	c.clips = append(c.clips, c.clip)
	c.clip = intersectPixelRect(c.clip, c.transformedRect(rect))
}

func (c *gpuCanvas) PopClip() {
	if len(c.clips) == 0 {
		return
	}
	c.clip = c.clips[len(c.clips)-1]
	c.clips = c.clips[:len(c.clips)-1]
}

func (c *gpuCanvas) BeginDamage(rect Rect) {
	region := intersectPixelRect(pixelRect{maxX: c.width, maxY: c.height}, c.transformedRect(rect))
	if region.maxX > region.minX && region.maxY > region.minY {
		c.addDamage(region)
	}
	c.damageDepth++
}

func (c *gpuCanvas) EndDamage() {
	if c.damageDepth > 0 {
		c.damageDepth--
	}
}

func (c *gpuCanvas) addDamage(region pixelRect) {
	for i := 0; i < len(c.damage); i++ {
		if pixelRectContains(c.damage[i], region) {
			return
		}
		if pixelRectContains(region, c.damage[i]) {
			copy(c.damage[i:], c.damage[i+1:])
			c.damage = c.damage[:len(c.damage)-1]
			i--
		}
	}
	c.damage = append(c.damage, region)
	if len(c.damage) > 64 {
		combined := c.damage[0]
		for i := 1; i < len(c.damage); i++ {
			combined = unionPixelRect(combined, c.damage[i])
		}
		c.damage = c.damage[:1]
		c.damage[0] = combined
	}
}

func sameDrawState(a, b drawCommand) bool {
	return a.kind == b.kind && a.blend == b.blend && a.sampling == b.sampling && a.clip == b.clip && a.image == b.image && a.color == b.color
}

func (c *gpuCanvas) appendCommand(command drawCommand, vertices []drawVertex) {
	if c.clip.maxX <= c.clip.minX || c.clip.maxY <= c.clip.minY || len(vertices) == 0 {
		return
	}
	command.clip = c.clip
	command.first = len(c.list.vertices)
	command.count = len(vertices)
	if len(c.list.commands) > 0 {
		last := &c.list.commands[len(c.list.commands)-1]
		if last.first+last.count == command.first && sameDrawState(*last, command) {
			c.list.vertices = append(c.list.vertices, vertices...)
			last.count += len(vertices)
			return
		}
	}
	c.list.vertices = append(c.list.vertices, vertices...)
	c.list.commands = append(c.list.commands, command)
}

func solidVertex(point Point) drawVertex { return drawVertex{x: point.X, y: point.Y} }

func (c *gpuCanvas) appendSolidTriangle(a, b, d Point, color Color) {
	c.appendCommand(drawCommand{kind: drawCommandSolid, blend: c.blend, image: -1, color: color}, []drawVertex{solidVertex(a), solidVertex(b), solidVertex(d)})
}

func (c *gpuCanvas) appendSolidQuad(a, b, d, e Point, color Color) {
	c.appendCommand(drawCommand{kind: drawCommandSolid, blend: c.blend, image: -1, color: color}, []drawVertex{
		solidVertex(a), solidVertex(b), solidVertex(d),
		solidVertex(a), solidVertex(d), solidVertex(e),
	})
}

func (c *gpuCanvas) Clear(color Color) {
	old := c.blend
	c.blend = BlendCopy
	a := Point{X: Scalar(c.clip.minX), Y: Scalar(c.clip.minY)}
	b := Point{X: Scalar(c.clip.maxX), Y: Scalar(c.clip.minY)}
	d := Point{X: Scalar(c.clip.maxX), Y: Scalar(c.clip.maxY)}
	e := Point{X: Scalar(c.clip.minX), Y: Scalar(c.clip.maxY)}
	c.appendSolidQuad(a, b, d, e, color)
	c.blend = old
}

func (c *gpuCanvas) DrawPoint(point Point, color Color) {
	point = c.transformPoint(point)
	x, y := scalarFloor(point.X), scalarFloor(point.Y)
	c.appendSolidQuad(Point{Scalar(x), Scalar(y)}, Point{Scalar(x + 1), Scalar(y)}, Point{Scalar(x + 1), Scalar(y + 1)}, Point{Scalar(x), Scalar(y + 1)}, color)
}

func (c *gpuCanvas) FillTriangle(a, b, d Point, color Color) {
	c.appendSolidTriangle(c.transformPoint(a), c.transformPoint(b), c.transformPoint(d), color)
}

func (c *gpuCanvas) FillRect(rect Rect, color Color) {
	if rect.Empty() {
		return
	}
	a := c.transformPoint(Point{X: rect.MinX, Y: rect.MinY})
	b := c.transformPoint(Point{X: rect.MaxX, Y: rect.MinY})
	d := c.transformPoint(Point{X: rect.MaxX, Y: rect.MaxY})
	e := c.transformPoint(Point{X: rect.MinX, Y: rect.MaxY})
	if c.transform.B == 0 && c.transform.C == 0 {
		minX, maxX, minY, maxY := pointBounds4(a, b, d, e)
		a = Point{X: Scalar(scalarFloor(minX)), Y: Scalar(scalarFloor(minY))}
		b = Point{X: Scalar(scalarCeil(maxX)), Y: a.Y}
		d = Point{X: b.X, Y: Scalar(scalarCeil(maxY))}
		e = Point{X: a.X, Y: d.Y}
	}
	c.appendSolidQuad(a, b, d, e, color)
}

func gpuSqrt(value Scalar) Scalar {
	if value <= 0 {
		return 0
	}
	result := value
	if result < 1 {
		result = 1
	}
	for i := 0; i < 8; i++ {
		result = (result + value/result) / 2
	}
	return result
}

func (c *gpuCanvas) DrawLine(a, b Point, width Scalar, color Color) {
	if width <= 0 {
		return
	}
	a = c.transformPoint(a)
	b = c.transformPoint(b)
	dx, dy := b.X-a.X, b.Y-a.Y
	length := gpuSqrt(dx*dx + dy*dy)
	half := width * c.deviceScale / 2
	if length == 0 {
		c.fillCirclePhysical(a, half, color)
		return
	}
	nx, ny := -dy*half/length, dx*half/length
	c.appendSolidQuad(Point{a.X + nx, a.Y + ny}, Point{b.X + nx, b.Y + ny}, Point{b.X - nx, b.Y - ny}, Point{a.X - nx, a.Y - ny}, color)
	c.fillCirclePhysical(a, half, color)
	c.fillCirclePhysical(b, half, color)
}

func (c *gpuCanvas) DrawPolyline(points []Point, width Scalar, color Color) {
	for i := 1; i < len(points); i++ {
		c.DrawLine(points[i-1], points[i], width, color)
	}
}

func (c *gpuCanvas) StrokeRect(rect Rect, width Scalar, color Color) {
	c.DrawLine(Point{rect.MinX, rect.MinY}, Point{rect.MaxX, rect.MinY}, width, color)
	c.DrawLine(Point{rect.MaxX, rect.MinY}, Point{rect.MaxX, rect.MaxY}, width, color)
	c.DrawLine(Point{rect.MaxX, rect.MaxY}, Point{rect.MinX, rect.MaxY}, width, color)
	c.DrawLine(Point{rect.MinX, rect.MaxY}, Point{rect.MinX, rect.MinY}, width, color)
}

func (c *gpuCanvas) fillCirclePhysical(center Point, radius Scalar, color Color) {
	if radius <= 0 {
		return
	}
	for i := 0; i < gpuCircleSegments; i++ {
		next := (i + 1) % gpuCircleSegments
		currentPoint, nextPoint := gpuCirclePoint(i), gpuCirclePoint(next)
		a := Point{center.X + currentPoint.X*radius, center.Y + currentPoint.Y*radius}
		b := Point{center.X + nextPoint.X*radius, center.Y + nextPoint.Y*radius}
		c.appendSolidTriangle(center, a, b, color)
	}
}

func (c *gpuCanvas) ellipseMask(width, height, stroke int) *Image {
	for i := 0; i < len(c.ellipses); i++ {
		entry := &c.ellipses[i]
		if entry.width == width && entry.height == height && entry.stroke == stroke {
			return entry.image
		}
	}
	pixels := make([]byte, width*height)
	widthSquared := width * width
	heightSquared := height * height
	outerLimit := widthSquared * heightSquared
	innerWidth := width - stroke*2
	innerHeight := height - stroke*2
	innerWidthSquared := innerWidth * innerWidth
	innerHeightSquared := innerHeight * innerHeight
	innerLimit := innerWidthSquared * innerHeightSquared
	for y := 0; y < height; y++ {
		dy := y*2 + 1 - height
		dySquared := dy * dy
		for x := 0; x < width; x++ {
			dx := x*2 + 1 - width
			dxSquared := dx * dx
			if dxSquared*heightSquared+dySquared*widthSquared > outerLimit {
				continue
			}
			if stroke > 0 && innerWidth > 0 && innerHeight > 0 {
				if dxSquared*innerHeightSquared+dySquared*innerWidthSquared <= innerLimit {
					continue
				}
			}
			pixels[y*width+x] = 255
		}
	}
	image := NewMask(width, height, pixels)
	c.ellipses = append(c.ellipses, ellipseMaskEntry{width: width, height: height, stroke: stroke, image: image})
	return image
}

func (c *gpuCanvas) drawEllipseMask(rect Rect, stroke Scalar, color Color) {
	if rect.Empty() {
		return
	}
	width := scalarCeil(rect.Width() * c.deviceScale)
	height := scalarCeil(rect.Height() * c.deviceScale)
	if width <= 0 || height <= 0 {
		return
	}
	strokePixels := 0
	if stroke > 0 {
		strokePixels = scalarCeil(stroke * c.deviceScale)
		if strokePixels < 1 {
			strokePixels = 1
		}
	}
	mask := c.ellipseMask(width, height, strokePixels)
	c.DrawImage(mask, R(0, 0, Scalar(width), Scalar(height)), rect, SamplingLinear, color)
}

func (c *gpuCanvas) FillEllipse(rect Rect, color Color) {
	c.drawEllipseMask(rect, 0, color)
}

func (c *gpuCanvas) StrokeEllipse(rect Rect, width Scalar, color Color) {
	if rect.Empty() || width <= 0 {
		return
	}
	if width*2 >= rect.Width() || width*2 >= rect.Height() {
		c.FillEllipse(rect, color)
		return
	}
	c.drawEllipseMask(rect, width, color)
}

func (c *gpuCanvas) FillConvexPolygon(points []Point, color Color) {
	if len(points) < 3 {
		return
	}
	a := c.transformPoint(points[0])
	for i := 2; i < len(points); i++ {
		c.appendSolidTriangle(a, c.transformPoint(points[i-1]), c.transformPoint(points[i]), color)
	}
}

func (c *gpuCanvas) FillPolygon(points []Point, rule FillRule, color Color) {
	if len(points) < 3 {
		return
	}
	var path Path
	path.MoveTo(points[0])
	for i := 1; i < len(points); i++ {
		path.LineTo(points[i])
	}
	path.Close()
	c.FillPath(&path, rule, color)
}

func insertPathCrossing(crossings []pathCrossing, crossing pathCrossing) []pathCrossing {
	crossings = append(crossings, crossing)
	for i := len(crossings) - 1; i > 0 && crossings[i].x < crossings[i-1].x; i-- {
		crossings[i], crossings[i-1] = crossings[i-1], crossings[i]
	}
	return crossings
}

func (c *gpuCanvas) appendPathSpan(y int, from, to Scalar, color Color) {
	minX := scalarCeil(from - 0.5)
	maxX := scalarCeil(to - 0.5)
	if minX < c.clip.minX {
		minX = c.clip.minX
	}
	if maxX > c.clip.maxX {
		maxX = c.clip.maxX
	}
	if maxX <= minX || y < c.clip.minY || y >= c.clip.maxY {
		return
	}
	c.appendSolidQuad(Point{Scalar(minX), Scalar(y)}, Point{Scalar(maxX), Scalar(y)}, Point{Scalar(maxX), Scalar(y + 1)}, Point{Scalar(minX), Scalar(y + 1)}, color)
}

func (c *gpuCanvas) FillPath(path *Path, rule FillRule, color Color) {
	if path == nil || len(path.points) < 3 {
		return
	}
	points := make([]Point, len(path.points))
	minY, maxY := Scalar(0), Scalar(0)
	for i := 0; i < len(path.points); i++ {
		points[i] = c.transformPoint(path.points[i])
		if i == 0 || points[i].Y < minY {
			minY = points[i].Y
		}
		if i == 0 || points[i].Y > maxY {
			maxY = points[i].Y
		}
	}
	startY, endY := scalarFloor(minY), scalarCeil(maxY)
	if startY < c.clip.minY {
		startY = c.clip.minY
	}
	if endY > c.clip.maxY {
		endY = c.clip.maxY
	}
	for y := startY; y < endY; y++ {
		c.crossings = c.crossings[:0]
		lineY := Scalar(y) + 0.5
		for contour := 0; contour < len(path.starts); contour++ {
			start, end := path.starts[contour], path.contourEnd(contour)
			for i := start; i < end; i++ {
				previous := end - 1
				if i > start {
					previous = i - 1
				}
				a, b := points[previous], points[i]
				if (a.Y <= lineY && b.Y > lineY) || (b.Y <= lineY && a.Y > lineY) {
					x := a.X + (lineY-a.Y)*(b.X-a.X)/(b.Y-a.Y)
					direction := -1
					if b.Y > a.Y {
						direction = 1
					}
					c.crossings = insertPathCrossing(c.crossings, pathCrossing{x: x, direction: direction})
				}
			}
		}
		if rule == FillEvenOdd {
			for i := 1; i < len(c.crossings); i += 2 {
				c.appendPathSpan(y, c.crossings[i-1].x, c.crossings[i].x, color)
			}
		} else {
			winding := 0
			span := Scalar(0)
			for i := 0; i < len(c.crossings); i++ {
				previous := winding
				winding += c.crossings[i].direction
				if previous == 0 && winding != 0 {
					span = c.crossings[i].x
				} else if previous != 0 && winding == 0 {
					c.appendPathSpan(y, span, c.crossings[i].x, color)
				}
			}
		}
	}
}

func (c *gpuCanvas) StrokePath(path *Path, width Scalar, color Color) {
	if path == nil {
		return
	}
	for contour := 0; contour < len(path.starts); contour++ {
		start, end := path.starts[contour], path.contourEnd(contour)
		for i := start + 1; i < end; i++ {
			c.DrawLine(path.points[i-1], path.points[i], width, color)
		}
		if end-start > 1 && path.closed[contour] {
			c.DrawLine(path.points[end-1], path.points[start], width, color)
		}
	}
}

func (c *gpuCanvas) imageIndex(image *Image) int {
	for i := 0; i < len(c.list.images); i++ {
		if c.list.images[i] == image {
			return i
		}
	}
	c.list.images = append(c.list.images, image)
	return len(c.list.images) - 1
}

func imageVertex(point Point, u, v Scalar) drawVertex {
	return drawVertex{x: point.X, y: point.Y, u: u, v: v}
}

func (c *gpuCanvas) appendImageQuad(image *Image, src Rect, a, b, d, e Point, sampling Sampling, tint Color) {
	kind := drawCommandRGBA
	if image.Format == PixelA8 {
		kind = drawCommandMask
	}
	index := c.imageIndex(image)
	c.appendCommand(drawCommand{kind: kind, blend: c.blend, sampling: sampling, image: index, color: tint}, []drawVertex{
		imageVertex(a, src.MinX, src.MinY), imageVertex(b, src.MaxX, src.MinY), imageVertex(d, src.MaxX, src.MaxY),
		imageVertex(a, src.MinX, src.MinY), imageVertex(d, src.MaxX, src.MaxY), imageVertex(e, src.MinX, src.MaxY),
	})
}

func (c *gpuCanvas) DrawImage(image *Image, src, dst Rect, sampling Sampling, tint Color) {
	if image == nil || src.Empty() || dst.Empty() {
		return
	}
	a := c.transformPoint(Point{X: dst.MinX, Y: dst.MinY})
	b := c.transformPoint(Point{X: dst.MaxX, Y: dst.MinY})
	d := c.transformPoint(Point{X: dst.MaxX, Y: dst.MaxY})
	e := c.transformPoint(Point{X: dst.MinX, Y: dst.MaxY})
	c.appendImageQuad(image, src, a, b, d, e, sampling, tint)
}

func (c *gpuCanvas) drawBuiltinGlyph(font *Font, position Point, r int, color Color) {
	scale := Scalar(font.Scale)
	for y := 0; y < 7; y++ {
		bits := glyphRow(r, y)
		for x := 0; x < 5; x++ {
			if bits&(1<<uint(4-x)) != 0 {
				c.FillRect(R(position.X+Scalar(x)*scale, position.Y+Scalar(y)*scale, scale, scale), color)
			}
		}
	}
}

func (c *gpuCanvas) DrawText(font *Font, baseline Point, text string, color Color) {
	c.drawText(font, baseline, text, nil, false, color)
}

func (c *gpuCanvas) drawText(font *Font, baseline Point, text string, data []byte, bytesMode bool, color Color) {
	if font == nil {
		return
	}
	lineHeight := font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap
	originX, x, y, previous := baseline.X, baseline.X, baseline.Y, -1
	rasterScale := c.textRasterScale()
	length := len(text)
	if bytesMode {
		length = len(data)
	}
	for at := 0; at < length; {
		r, size := 0, 0
		if bytesMode {
			r, size = nextUTF8Bytes(data, at)
		} else {
			r, size = nextUTF8(text, at)
		}
		at += size
		if r == 10 {
			x, y, previous = originX, y+lineHeight, -1
		} else if r == 9 {
			if font.trueType != nil {
				x += font.cachedGlyph(' ').advance * 4
			} else {
				x += Scalar(6*font.Scale) * 4
			}
			previous = -1
		} else if font.trueType != nil {
			glyph := font.cachedGlyphAtScale(r, rasterScale)
			x += font.kern(previous, glyph.index)
			if glyph.mask != nil {
				origin := c.transformPoint(Point{X: x, Y: y})
				drawX := Scalar(scalarFloor(origin.X + glyph.xOffset + 0.5))
				drawY := Scalar(scalarFloor(origin.Y + glyph.yOffset + 0.5))
				src := R(0, 0, Scalar(glyph.mask.Width), Scalar(glyph.mask.Height))
				c.appendImageQuad(glyph.mask, src, Point{drawX, drawY}, Point{drawX + Scalar(glyph.mask.Width), drawY}, Point{drawX + Scalar(glyph.mask.Width), drawY + Scalar(glyph.mask.Height)}, Point{drawX, drawY + Scalar(glyph.mask.Height)}, SamplingNearest, color)
			}
			x += glyph.advance
			previous = glyph.index
		} else {
			c.drawBuiltinGlyph(font, Point{X: x, Y: y - font.Metrics.Ascent}, r, color)
			x += Scalar(6 * font.Scale)
		}
	}
}

func (c *gpuCanvas) textRasterScale() int {
	scale := c.transform.A * c.deviceScale
	if c.transform.B == 0 && c.transform.C == 0 && c.transform.A == c.transform.D && scale >= 1 && scale <= 4 && Scalar(int(scale)) == scale {
		return int(scale)
	}
	return 1
}

func (c *gpuCanvas) DrawTextBytes(font *Font, baseline Point, text []byte, color Color) {
	c.drawText(font, baseline, "", text, true, color)
}

func (c *gpuCanvas) DrawGlyphRun(origin Point, glyphs []Glyph, color Color) {
	for i := 0; i < len(glyphs); i++ {
		glyph := glyphs[i]
		if glyph.Mask != nil {
			c.DrawImage(glyph.Mask, glyph.Source, R(origin.X+glyph.X, origin.Y+glyph.Y, glyph.Source.Width(), glyph.Source.Height()), SamplingNearest, color)
		}
	}
}
