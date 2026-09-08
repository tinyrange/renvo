package graphics

// Rotation is clockwise display orientation, independent of canvas transforms.
// Coordinates/damage are logical; Pixels/Stride always describe native storage.
type Rotation byte

const (
	Rotation0 Rotation = iota
	Rotation90
	Rotation180
	Rotation270
)

// NewRotatedSurface allocates only the native buffer, already in scanout order.
func NewRotatedSurface(nativeWidth, nativeHeight int, format PixelFormat, rotation Rotation) *Surface {
	if nativeWidth < 0 || nativeHeight < 0 || rotation > Rotation270 || pixelFormatBytes(format) == 0 {
		return nil
	}
	s := allocSurface()
	s.resetFormat(nativeWidth, nativeHeight, format)
	s.orient(rotation)
	return s
}

// NewRotatedSurfaceBuffer wraps native storage without clearing it. Repaint
// before presenting if its contents are undefined.
func NewRotatedSurfaceBuffer(nativeWidth, nativeHeight int, format PixelFormat, rotation Rotation, pixels []byte) *Surface {
	if rotation > Rotation270 {
		return nil
	}
	s := NewSurfaceBufferFormatPreserve(nativeWidth, nativeHeight, format, pixels)
	if s != nil {
		s.orient(rotation)
	}
	return s
}

func (s *Surface) orient(rotation Rotation) {
	s.rotation = rotation
	if rotation == Rotation90 || rotation == Rotation270 {
		s.Width, s.Height = s.Height, s.Width
	}
	s.clip = pixelRect{maxX: s.Width, maxY: s.Height}
	s.ResetDirty()
	if s.Width > 0 && s.Height > 0 {
		s.markDirtyRect(s.clip)
	}
}

func (s *Surface) Rotation() Rotation { return s.rotation }

// SetRotation changes layout without moving/allocating pixels. Repaint before
// presenting. It resets clips/transforms; logical contents are not preserved.
func (s *Surface) SetRotation(rotation Rotation) bool {
	if s == nil || rotation > Rotation270 {
		return false
	}
	if s.rotation == rotation {
		return true
	}
	width, height := s.NativeWidth(), s.NativeHeight()
	s.Width, s.Height = width, height
	s.clips = s.clips[:0]
	s.transforms = s.transforms[:0]
	s.transformComplexes = s.transformComplexes[:0]
	s.ResetTransform()
	s.orient(rotation)
	s.revision++
	return true
}

func (s *Surface) NativeWidth() int {
	if s.rotation == Rotation90 || s.rotation == Rotation270 {
		return s.Height
	}
	return s.Width
}

func (s *Surface) NativeHeight() int {
	if s.rotation == Rotation90 || s.rotation == Rotation270 {
		return s.Width
	}
	return s.Height
}

func (s *Surface) nativePoint(x, y int) (int, int) {
	switch s.rotation {
	case Rotation90:
		return y, s.Width - 1 - x
	case Rotation180:
		return s.Width - 1 - x, s.Height - 1 - y
	case Rotation270:
		return s.Height - 1 - y, x
	}
	return x, y
}

// PixelOffset returns the native byte offset of an in-bounds logical pixel.
func (s *Surface) PixelOffset(x, y int) int {
	x, y = s.nativePoint(x, y)
	return y*s.Stride + x*pixelFormatBytes(s.Format)
}

func (s *Surface) nativeRect(r pixelRect) pixelRect {
	switch s.rotation {
	case Rotation90:
		return pixelRect{r.minY, s.Width - r.maxX, r.maxY, s.Width - r.minX}
	case Rotation180:
		return pixelRect{s.Width - r.maxX, s.Height - r.maxY, s.Width - r.minX, s.Height - r.minY}
	case Rotation270:
		return pixelRect{s.Height - r.maxY, r.minX, s.Height - r.minY, r.maxX}
	}
	return r
}

// NativeRect maps an in-bounds logical rectangle to scanout coordinates.
func (s *Surface) NativeRect(r Rect) Rect {
	p := s.nativeRect(pixelRect{scalarFloor(r.MinX), scalarFloor(r.MinY), scalarCeil(r.MaxX), scalarCeil(r.MaxY)})
	return Rect{Scalar(p.minX), Scalar(p.minY), Scalar(p.maxX), Scalar(p.maxY)}
}

// CopyPixels moves a logical rectangle, handling overlap and marking damage.
// It ignores transforms/clips. Invalid bounds leave storage unchanged.
func (s *Surface) CopyPixels(dstX, dstY, srcX, srcY, width, height int) bool {
	if width <= 0 || height <= 0 || srcX < 0 || srcY < 0 || dstX < 0 || dstY < 0 ||
		width > s.Width || height > s.Height || srcX > s.Width-width || dstX > s.Width-width ||
		srcY > s.Height-height || dstY > s.Height-height {
		return false
	}
	damage := pixelRect{dstX, dstY, dstX + width, dstY + height}
	source := s.nativeRect(pixelRect{srcX, srcY, srcX + width, srcY + height})
	dest := s.nativeRect(damage)
	bytes := pixelFormatBytes(s.Format)
	rowBytes := (source.maxX - source.minX) * bytes
	rows := source.maxY - source.minY
	for i := 0; i < rows; i++ {
		row := i
		if dest.minY > source.minY {
			row = rows - 1 - i
		}
		from := (source.minY+row)*s.Stride + source.minX*bytes
		to := (dest.minY+row)*s.Stride + dest.minX*bytes
		copy(s.Pixels[to:to+rowBytes], s.Pixels[from:from+rowBytes])
	}
	s.MarkUpdated(Rect{Scalar(dstX), Scalar(dstY), Scalar(dstX + width), Scalar(dstY + height)})
	return true
}
