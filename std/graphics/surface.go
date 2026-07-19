package graphics

type pixelRect struct {
	minX int
	minY int
	maxX int
	maxY int
}

// Surface is a top-down, tightly packed, premultiplied RGBA8 render target.
type Surface struct {
	Width  int
	Height int
	Stride int
	Pixels []byte
	Format PixelFormat

	blend              BlendMode
	clip               pixelRect
	clips              []pixelRect
	dirty              pixelRect
	dirtyValid         bool
	dirtyRects         []pixelRect
	damageDepth        int
	transformA         Scalar
	transformB         Scalar
	transformC         Scalar
	transformD         Scalar
	transformTX        Scalar
	transformTY        Scalar
	transforms         []Mat2x3
	transformComplex   bool
	transformComplexes []bool
}

// Image is an RGBA8 pixel resource. It aliases Surface so off-screen render
// targets and uploaded images share one portable representation.
type Image = Surface

func NewImage(width, height int, pixels []byte) *Image {
	return NewImageFormat(width, height, PixelRGBA8, pixels)
}

func NewImageFormat(width, height int, format PixelFormat, pixels []byte) *Image {
	image := NewSurface(width, height)
	image.Format = format
	if format == PixelA8 {
		image.Stride = width
		image.Pixels = make([]byte, image.Stride*height)
	}
	limit := len(image.Pixels)
	if len(pixels) < limit {
		limit = len(pixels)
	}
	for i := 0; i < limit; i++ {
		image.Pixels[i] = pixels[i]
	}
	return image
}

func NewMask(width, height int, pixels []byte) *Image {
	return NewImageFormat(width, height, PixelA8, pixels)
}

func (s *Surface) Destroy() {
	s.Width = 0
	s.Height = 0
	s.Stride = 0
	s.Pixels = nil
}

func (s *Surface) UpdateImage(rect Rect, pixels []byte) {
	minX, minY := scalarFloor(rect.MinX), scalarFloor(rect.MinY)
	maxX, maxY := scalarCeil(rect.MaxX), scalarCeil(rect.MaxY)
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX > s.Width {
		maxX = s.Width
	}
	if maxY > s.Height {
		maxY = s.Height
	}
	pixelSize := 4
	if s.Format == PixelA8 {
		pixelSize = 1
	}
	i := 0
	for y := minY; y < maxY; y++ {
		for x := minX; x < maxX; x++ {
			if i+pixelSize > len(pixels) {
				return
			}
			o := y*s.Stride + x*pixelSize
			for channel := 0; channel < pixelSize; channel++ {
				s.Pixels[o+channel] = pixels[i+channel]
			}
			i += pixelSize
			s.markDirty(x, y)
		}
	}
}

func (s *Surface) DrawPolyline(points []Point, width Scalar, color Color) {
	for i := 1; i < len(points); i++ {
		s.DrawLine(points[i-1], points[i], width, color)
	}
}

func (s *Surface) FillConvexPolygon(points []Point, color Color) {
	if len(points) < 3 {
		return
	}
	var path Path
	path.MoveTo(points[0])
	for i := 1; i < len(points); i++ {
		path.LineTo(points[i])
	}
	path.Close()
	s.FillPath(&path, FillNonZero, color)
}

func NewSurface(width, height int) *Surface {
	s := allocSurface()
	s.reset(width, height)
	return s
}

func (s *Surface) reset(width, height int) {
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}
	s.Width = width
	s.Height = height
	s.Stride = width * 4
	s.Pixels = make([]byte, s.Stride*height)
	s.Format = PixelRGBA8
	s.blend = BlendSourceOver
	s.clip = pixelRect{maxX: width, maxY: height}
	s.dirty = pixelRect{maxX: width, maxY: height}
	s.dirtyValid = width > 0 && height > 0
	s.dirtyRects = s.dirtyRects[:0]
	if s.dirtyValid {
		s.dirtyRects = append(s.dirtyRects, s.dirty)
	}
	s.damageDepth = 0
	s.ResetTransform()
	s.clips = nil
	s.transforms = nil
	s.transformComplexes = nil
}

func (s *Surface) Resize(width, height int) {
	s.reset(width, height)
}

func (s *Surface) DirtyRect() (Rect, bool) {
	if s == nil || !s.dirtyValid {
		return Rect{}, false
	}
	return Rect{MinX: Scalar(s.dirty.minX), MinY: Scalar(s.dirty.minY), MaxX: Scalar(s.dirty.maxX), MaxY: Scalar(s.dirty.maxY)}, true
}

// DirtyRects returns the precise damage regions registered since the previous
// presentation. Ordinary drawing without a damage scope falls back to one
// bounding rectangle; retained views use BeginDamage to preserve disjoint
// regions all the way to native presentation.
func (s *Surface) DirtyRects() []Rect {
	if s == nil || !s.dirtyValid {
		return nil
	}
	out := make([]Rect, len(s.dirtyRects))
	for i := 0; i < len(s.dirtyRects); i++ {
		region := s.dirtyRects[i]
		out[i] = Rect{MinX: Scalar(region.minX), MinY: Scalar(region.minY), MaxX: Scalar(region.maxX), MaxY: Scalar(region.maxY)}
	}
	return out
}

// BeginDamage declares the exact clipped area a retained view is about to
// repaint. Calls may be nested; every BeginDamage must have a matching
// EndDamage.
func (s *Surface) BeginDamage(rect Rect) {
	if s == nil {
		return
	}
	region := pixelRect{minX: scalarFloor(rect.MinX), minY: scalarFloor(rect.MinY), maxX: scalarCeil(rect.MaxX), maxY: scalarCeil(rect.MaxY)}
	region = intersectPixelRect(pixelRect{maxX: s.Width, maxY: s.Height}, region)
	if region.maxX > region.minX && region.maxY > region.minY {
		s.markDirtyRect(region)
	}
	s.damageDepth++
}

func (s *Surface) EndDamage() {
	if s != nil && s.damageDepth > 0 {
		s.damageDepth--
	}
}

func (s *Surface) ResetDirty() {
	if s == nil {
		return
	}
	s.dirtyValid = false
	s.dirtyRects = s.dirtyRects[:0]
	s.damageDepth = 0
}

func (s *Surface) markDirty(x, y int) {
	if !s.dirtyValid {
		s.dirty = pixelRect{minX: x, minY: y, maxX: x + 1, maxY: y + 1}
		s.dirtyValid = true
		if s.damageDepth == 0 {
			s.dirtyRects = append(s.dirtyRects, s.dirty)
		}
		return
	}
	if x < s.dirty.minX {
		s.dirty.minX = x
	}
	if y < s.dirty.minY {
		s.dirty.minY = y
	}
	if x+1 > s.dirty.maxX {
		s.dirty.maxX = x + 1
	}
	if y+1 > s.dirty.maxY {
		s.dirty.maxY = y + 1
	}
	if s.damageDepth == 0 {
		// Immediate-mode callers do not supply damage boundaries. Retain the
		// historical bounding rectangle rather than recording one region per
		// pixel.
		if len(s.dirtyRects) == 0 {
			s.dirtyRects = append(s.dirtyRects, s.dirty)
		} else {
			s.dirtyRects = s.dirtyRects[:1]
			s.dirtyRects[0] = s.dirty
		}
	}
}

func (s *Surface) markDirtyRect(region pixelRect) {
	if !s.dirtyValid {
		s.dirty = region
		s.dirtyValid = true
	} else {
		s.dirty = unionPixelRect(s.dirty, region)
	}
	for i := 0; i < len(s.dirtyRects); i++ {
		if pixelRectContains(s.dirtyRects[i], region) {
			return
		}
		if pixelRectContains(region, s.dirtyRects[i]) {
			copy(s.dirtyRects[i:], s.dirtyRects[i+1:])
			s.dirtyRects = s.dirtyRects[:len(s.dirtyRects)-1]
			i--
		}
	}
	s.dirtyRects = append(s.dirtyRects, region)
	if len(s.dirtyRects) > 64 {
		s.dirtyRects = s.dirtyRects[:1]
		s.dirtyRects[0] = s.dirty
	}
}

func unionPixelRect(a, b pixelRect) pixelRect {
	if b.minX < a.minX {
		a.minX = b.minX
	}
	if b.minY < a.minY {
		a.minY = b.minY
	}
	if b.maxX > a.maxX {
		a.maxX = b.maxX
	}
	if b.maxY > a.maxY {
		a.maxY = b.maxY
	}
	return a
}

func pixelRectContains(outer, inner pixelRect) bool {
	return outer.minX <= inner.minX && outer.minY <= inner.minY && outer.maxX >= inner.maxX && outer.maxY >= inner.maxY
}

func (s *Surface) SetBlendMode(mode BlendMode) { s.blend = mode }
func (s *Surface) SetTransform(m *Mat2x3) {
	if m == nil {
		s.ResetTransform()
		return
	}
	s.transformA = linearArgumentScalar(m.A)
	s.transformB = linearArgumentScalar(m.B)
	s.transformC = linearArgumentScalar(m.C)
	s.transformD = linearArgumentScalar(m.D)
	s.transformTX = matrixScalar(m.TX)
	s.transformTY = matrixScalar(m.TY)
	s.transformComplex = true
}

func (s *Surface) SetAffine(a, b, c, d, tx, ty Scalar) {
	s.transformA = linearArgumentScalar(a)
	s.transformB = linearArgumentScalar(b)
	s.transformC = linearArgumentScalar(c)
	s.transformD = linearArgumentScalar(d)
	s.transformTX = tx
	s.transformTY = ty
	s.transformComplex = true
}

func (s *Surface) SetLinear(a, b, c, d Scalar) {
	s.SetAxisX(a, b)
	s.SetAxisY(c, d)
}

func (s *Surface) SetAxisX(x, y Scalar) {
	s.transformA = linearArgumentScalar(x)
	s.transformB = linearArgumentScalar(y)
	s.transformComplex = true
}

func (s *Surface) SetAxisY(x, y Scalar) {
	s.transformC = linearArgumentScalar(x)
	s.transformD = linearArgumentScalar(y)
	s.transformComplex = true
}

func (s *Surface) SetOffset(tx, ty Scalar) {
	s.transformTX = tx
	s.transformTY = ty
}

func (s *Surface) ResetTransform() {
	s.transformA = 1
	s.transformB = 0
	s.transformC = 0
	s.transformD = 1
	s.transformTX = 0
	s.transformTY = 0
	s.transformComplex = false
}

func (s *Surface) SetTranslation(x, y Scalar) {
	s.ResetTransform()
	s.transformTX = x
	s.transformTY = y
}

func (s *Surface) transformIsIdentity() bool {
	return s.transformA == 1.0 && s.transformB == 0.0 && s.transformC == 0.0 && s.transformD == 1.0 && s.transformTX == 0.0 && s.transformTY == 0.0
}

func (s *Surface) transformPoint(p Point) Point {
	s.transformPointInPlace(&p)
	return p
}

func (s *Surface) TransformPoint(point Point) Point { return s.transformPoint(point) }

func (s *Surface) transformPointInPlace(p *Point) {
	x, y := p.X, p.Y
	p.X = s.transformA*x + s.transformC*y + s.transformTX
	p.Y = s.transformB*x + s.transformD*y + s.transformTY
}

func (s *Surface) PushTransform() {
	s.transforms = append(s.transforms, Mat2x3{A: s.transformA, B: s.transformB, C: s.transformC, D: s.transformD, TX: s.transformTX, TY: s.transformTY})
	s.transformComplexes = append(s.transformComplexes, s.transformComplex)
}

func (s *Surface) PopTransform() {
	if len(s.transforms) == 0 {
		return
	}
	last := s.transforms[len(s.transforms)-1]
	s.transforms = s.transforms[:len(s.transforms)-1]
	s.transformA = last.A
	s.transformB = last.B
	s.transformC = last.C
	s.transformD = last.D
	s.transformTX = last.TX
	s.transformTY = last.TY
	s.transformComplex = s.transformComplexes[len(s.transformComplexes)-1]
	s.transformComplexes = s.transformComplexes[:len(s.transformComplexes)-1]
}

func scalarFloor(v Scalar) int {
	i := int(v)
	if Scalar(i) > v {
		return i - 1
	}
	return i
}

func scalarCeil(v Scalar) int {
	i := int(v)
	if Scalar(i) < v {
		return i + 1
	}
	return i
}

func intersectPixelRect(a, b pixelRect) pixelRect {
	if b.minX > a.minX {
		a.minX = b.minX
	}
	if b.minY > a.minY {
		a.minY = b.minY
	}
	if b.maxX < a.maxX {
		a.maxX = b.maxX
	}
	if b.maxY < a.maxY {
		a.maxY = b.maxY
	}
	if a.maxX < a.minX {
		a.maxX = a.minX
	}
	if a.maxY < a.minY {
		a.maxY = a.minY
	}
	return a
}

func (s *Surface) PushClipRect(r Rect) {
	s.clips = append(s.clips, s.clip)
	next := pixelRect{minX: scalarFloor(r.MinX), minY: scalarFloor(r.MinY), maxX: scalarCeil(r.MaxX), maxY: scalarCeil(r.MaxY)}
	s.clip = intersectPixelRect(s.clip, next)
}

func (s *Surface) PopClip() {
	if len(s.clips) == 0 {
		return
	}
	s.clip = s.clips[len(s.clips)-1]
	s.clips = s.clips[:len(s.clips)-1]
}

func (s *Surface) putPixel(x, y int, c Color) {
	if x < s.clip.minX || x >= s.clip.maxX || y < s.clip.minY || y >= s.clip.maxY {
		return
	}
	o := y*s.Stride + x*4
	s.markDirty(x, y)
	if s.blend == BlendCopy || c.A == 255 {
		s.Pixels[o], s.Pixels[o+1], s.Pixels[o+2], s.Pixels[o+3] = c.R, c.G, c.B, c.A
		return
	}
	inv := 255 - int(c.A)
	s.Pixels[o] = byte(int(c.R) + (int(s.Pixels[o])*inv+127)/255)
	s.Pixels[o+1] = byte(int(c.G) + (int(s.Pixels[o+1])*inv+127)/255)
	s.Pixels[o+2] = byte(int(c.B) + (int(s.Pixels[o+2])*inv+127)/255)
	s.Pixels[o+3] = byte(int(c.A) + (int(s.Pixels[o+3])*inv+127)/255)
}

func (s *Surface) Clear(c Color) {
	old := s.blend
	s.blend = BlendCopy
	for y := s.clip.minY; y < s.clip.maxY; y++ {
		for x := s.clip.minX; x < s.clip.maxX; x++ {
			s.putPixel(x, y, c)
		}
	}
	s.blend = old
}

func edge(a, b, p Point) Scalar { return (b.X-a.X)*(p.Y-a.Y) - (b.Y-a.Y)*(p.X-a.X) }

func topLeftEdge(a, b Point) bool {
	dy := b.Y - a.Y
	dx := b.X - a.X
	return dy < 0.0 || (dy == 0.0 && dx > 0.0)
}

func (s *Surface) FillTriangle(a, b, c Point, color Color) {
	s.transformPointInPlace(&a)
	s.transformPointInPlace(&b)
	s.transformPointInPlace(&c)
	area := edge(a, b, c)
	if area == 0.0 {
		return
	}
	if area < 0.0 {
		b, c = c, b
		area = -area
	}
	minX, maxX := a.X, a.X
	minY, maxY := a.Y, a.Y
	if b.X < minX {
		minX = b.X
	}
	if c.X < minX {
		minX = c.X
	}
	if b.X > maxX {
		maxX = b.X
	}
	if c.X > maxX {
		maxX = c.X
	}
	if b.Y < minY {
		minY = b.Y
	}
	if c.Y < minY {
		minY = c.Y
	}
	if b.Y > maxY {
		maxY = b.Y
	}
	if c.Y > maxY {
		maxY = c.Y
	}
	loX, hiX := scalarFloor(minX), scalarCeil(maxX)
	loY, hiY := scalarFloor(minY), scalarCeil(maxY)
	for y := loY; y < hiY; y++ {
		for x := loX; x < hiX; x++ {
			p := Point{X: Scalar(x) + 0.5, Y: Scalar(y) + 0.5}
			e0, e1, e2 := edge(a, b, p), edge(b, c, p), edge(c, a, p)
			inside0 := e0 > 0.0 || (e0 == 0.0 && topLeftEdge(a, b))
			inside1 := e1 > 0.0 || (e1 == 0.0 && topLeftEdge(b, c))
			inside2 := e2 > 0.0 || (e2 == 0.0 && topLeftEdge(c, a))
			if inside0 && inside1 && inside2 {
				s.putPixel(x, y, color)
			}
		}
	}
}

func (s *Surface) FillRect(r Rect, color Color) {
	a := Point{X: r.MinX, Y: r.MinY}
	b := Point{X: r.MaxX, Y: r.MinY}
	c := Point{X: r.MaxX, Y: r.MaxY}
	d := Point{X: r.MinX, Y: r.MaxY}
	s.transformPointInPlace(&a)
	s.transformPointInPlace(&b)
	s.transformPointInPlace(&c)
	s.transformPointInPlace(&d)
	minX, maxX, minY, maxY := a.X, a.X, a.Y, a.Y
	points := []Point{b, c, d}
	for i := 0; i < len(points); i++ {
		point := points[i]
		if point.X < minX {
			minX = point.X
		}
		if point.X > maxX {
			maxX = point.X
		}
		if point.Y < minY {
			minY = point.Y
		}
		if point.Y > maxY {
			maxY = point.Y
		}
	}
	for y := scalarFloor(minY); y < scalarCeil(maxY); y++ {
		for x := scalarFloor(minX); x < scalarCeil(maxX); x++ {
			point := Point{X: Scalar(x) + 0.5, Y: Scalar(y) + 0.5}
			e0 := edge(a, b, point)
			e1 := edge(b, c, point)
			e2 := edge(c, d, point)
			e3 := edge(d, a, point)
			hasNegative := e0 < 0.0 || e1 < 0.0 || e2 < 0.0 || e3 < 0.0
			hasPositive := e0 > 0.0 || e1 > 0.0 || e2 > 0.0 || e3 > 0.0
			if !hasNegative || !hasPositive {
				s.putPixel(x, y, color)
			}
		}
	}
}

func (s *Surface) StrokeRect(r Rect, width Scalar, color Color) {
	s.DrawLine(Point{r.MinX, r.MinY}, Point{r.MaxX, r.MinY}, width, color)
	s.DrawLine(Point{r.MaxX, r.MinY}, Point{r.MaxX, r.MaxY}, width, color)
	s.DrawLine(Point{r.MaxX, r.MaxY}, Point{r.MinX, r.MaxY}, width, color)
	s.DrawLine(Point{r.MinX, r.MaxY}, Point{r.MinX, r.MinY}, width, color)
}

func (s *Surface) DrawLine(a, b Point, width Scalar, color Color) {
	s.transformPointInPlace(&a)
	s.transformPointInPlace(&b)
	if width <= 0.0 {
		return
	}
	half := width / 2.0
	minX, maxX := a.X, b.X
	minY, maxY := a.Y, b.Y
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	if minY > maxY {
		minY, maxY = maxY, minY
	}
	dx, dy := b.X-a.X, b.Y-a.Y
	length2 := dx*dx + dy*dy
	half2 := half * half
	for y := scalarFloor(minY - half); y < scalarCeil(maxY+half); y++ {
		for x := scalarFloor(minX - half); x < scalarCeil(maxX+half); x++ {
			p := Point{Scalar(x) + 0.5, Scalar(y) + 0.5}
			fromAX, fromAY := p.X-a.X, p.Y-a.Y
			dot := fromAX*dx + fromAY*dy
			cross := dx*fromAY - dy*fromAX
			inside := false
			if cross == 0.0 && dot >= 0.0 && dot <= length2 {
				inside = true
			} else if length2 == 0.0 || dot <= 0.0 {
				inside = fromAX*fromAX+fromAY*fromAY <= half2
			} else if dot >= length2 {
				fromBX, fromBY := p.X-b.X, p.Y-b.Y
				inside = fromBX*fromBX+fromBY*fromBY <= half2
			} else {
				inside = cross*cross <= half2*length2
			}
			if inside {
				s.putPixel(x, y, color)
			}
		}
	}
}

func tintColor(src, tint Color) Color {
	return Color{R: byte((int(src.R)*int(tint.R) + 127) / 255), G: byte((int(src.G)*int(tint.G) + 127) / 255), B: byte((int(src.B)*int(tint.B) + 127) / 255), A: byte((int(src.A)*int(tint.A) + 127) / 255)}
}

func (image *Surface) imagePixel(x, y int) Color {
	if x < 0 || y < 0 || x >= image.Width || y >= image.Height {
		return Color{}
	}
	if image.Format == PixelA8 {
		a := image.Pixels[y*image.Stride+x]
		return Color{R: a, G: a, B: a, A: a}
	}
	o := y*image.Stride + x*4
	return Color{image.Pixels[o], image.Pixels[o+1], image.Pixels[o+2], image.Pixels[o+3]}
}

func colorLerp(a, b Color, amount Scalar) Color {
	weight := int(amount * 256)
	if weight < 0 {
		weight = 0
	}
	if weight > 256 {
		weight = 256
	}
	inverse := 256 - weight
	return Color{
		R: byte((int(a.R)*inverse + int(b.R)*weight + 128) / 256),
		G: byte((int(a.G)*inverse + int(b.G)*weight + 128) / 256),
		B: byte((int(a.B)*inverse + int(b.B)*weight + 128) / 256),
		A: byte((int(a.A)*inverse + int(b.A)*weight + 128) / 256),
	}
}

func colorLerp256(a, b Color, weight int) Color {
	if weight < 0 {
		weight = 0
	}
	if weight > 256 {
		weight = 256
	}
	inverse := 256 - weight
	return Color{
		R: byte((int(a.R)*inverse + int(b.R)*weight + 128) / 256),
		G: byte((int(a.G)*inverse + int(b.G)*weight + 128) / 256),
		B: byte((int(a.B)*inverse + int(b.B)*weight + 128) / 256),
		A: byte((int(a.A)*inverse + int(b.A)*weight + 128) / 256),
	}
}

func fixedFloor256(value int) int {
	if value < 0 {
		return -((-value + 255) / 256)
	}
	return value / 256
}

func (image *Surface) sampleImageFixed(x, y int, sampling Sampling) Color {
	if sampling == SamplingLinear {
		x0, y0 := fixedFloor256(x), fixedFloor256(y)
		top := colorLerp256(image.imagePixel(x0, y0), image.imagePixel(x0+1, y0), x-x0*256)
		bottom := colorLerp256(image.imagePixel(x0, y0+1), image.imagePixel(x0+1, y0+1), x-x0*256)
		return colorLerp256(top, bottom, y-y0*256)
	}
	return image.imagePixel(fixedFloor256(x+128), fixedFloor256(y+128))
}

func (image *Surface) sampleImage(x, y Scalar, sampling Sampling) Color {
	if sampling == SamplingLinear {
		x0, y0 := scalarFloor(x), scalarFloor(y)
		top := colorLerp(image.imagePixel(x0, y0), image.imagePixel(x0+1, y0), x-Scalar(x0))
		bottom := colorLerp(image.imagePixel(x0, y0+1), image.imagePixel(x0+1, y0+1), x-Scalar(x0))
		return colorLerp(top, bottom, y-Scalar(y0))
	}
	return image.imagePixel(scalarFloor(x+0.5), scalarFloor(y+0.5))
}

func scalarIsInteger(value Scalar) bool { return Scalar(int(value)) == value }

func (s *Surface) drawImageAxisAligned(image *Surface, src, dst Rect, sampling Sampling, tint Color) bool {
	if !s.transformIsIdentity() || !scalarIsInteger(src.MinX) || !scalarIsInteger(src.MinY) ||
		!scalarIsInteger(src.MaxX) || !scalarIsInteger(src.MaxY) || !scalarIsInteger(dst.MinX) ||
		!scalarIsInteger(dst.MinY) || !scalarIsInteger(dst.MaxX) || !scalarIsInteger(dst.MaxY) {
		return false
	}
	srcMinX, srcMinY := int(src.MinX), int(src.MinY)
	srcWidth, srcHeight := int(src.Width()), int(src.Height())
	dstMinX, dstMinY := int(dst.MinX), int(dst.MinY)
	dstWidth, dstHeight := int(dst.Width()), int(dst.Height())
	if srcWidth <= 0 || srcHeight <= 0 || dstWidth <= 0 || dstHeight <= 0 {
		return true
	}
	for y := dstMinY; y < dstMinY+dstHeight; y++ {
		sampleY := srcMinY*256 + ((2*(y-dstMinY)+1)*srcHeight*128)/dstHeight - 128
		for x := dstMinX; x < dstMinX+dstWidth; x++ {
			sampleX := srcMinX*256 + ((2*(x-dstMinX)+1)*srcWidth*128)/dstWidth - 128
			s.putPixel(x, y, tintColor(image.sampleImageFixed(sampleX, sampleY, sampling), tint))
		}
	}
	return true
}

// DrawImage draws a cropped image through the current affine transform.
func (s *Surface) DrawImage(image *Surface, src, dst Rect, sampling Sampling, tint Color) {
	if image == nil || src.Empty() || dst.Empty() {
		return
	}
	if s.drawImageAxisAligned(image, src, dst, sampling, tint) {
		return
	}
	a := Point{X: dst.MinX, Y: dst.MinY}
	b := Point{X: dst.MaxX, Y: dst.MinY}
	c := Point{X: dst.MaxX, Y: dst.MaxY}
	d := Point{X: dst.MinX, Y: dst.MaxY}
	s.transformPointInPlace(&a)
	s.transformPointInPlace(&b)
	s.transformPointInPlace(&c)
	s.transformPointInPlace(&d)
	minX, maxX, minY, maxY := a.X, a.X, a.Y, a.Y
	points := []Point{b, c, d}
	for i := 0; i < len(points); i++ {
		p := points[i]
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	determinant := s.transformA*s.transformD - s.transformB*s.transformC
	if determinant == 0.0 {
		return
	}
	for y := scalarFloor(minY); y < scalarCeil(maxY); y++ {
		for x := scalarFloor(minX); x < scalarCeil(maxX); x++ {
			sx, sy := Scalar(x)+0.5-s.transformTX, Scalar(y)+0.5-s.transformTY
			localX := (s.transformD*sx - s.transformC*sy) / determinant
			localY := (-s.transformB*sx + s.transformA*sy) / determinant
			u := (localX - dst.MinX) / dst.Width()
			v := (localY - dst.MinY) / dst.Height()
			if u < 0.0 || u >= 1.0 || v < 0.0 || v >= 1.0 {
				continue
			}
			sampleX := src.MinX + u*src.Width() - 0.5
			sampleY := src.MinY + v*src.Height() - 0.5
			s.putPixel(x, y, tintColor(image.sampleImage(sampleX, sampleY, sampling), tint))
		}
	}
}
