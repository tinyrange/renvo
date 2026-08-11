package graphics

// CanvasState is the portable drawing-state contract shared by software and
// accelerated canvases. State is local to one canvas and stacks are balanced
// by the caller.
type CanvasState interface {
	SetBlendMode(BlendMode)
	SetTransform(*Mat2x3)
	SetAffine(a, b, c, d, tx, ty Scalar)
	SetLinear(a, b, c, d Scalar)
	SetAxisX(x, y Scalar)
	SetAxisY(x, y Scalar)
	SetOffset(tx, ty Scalar)
	SetTranslation(x, y Scalar)
	ResetTransform()
	PushTransform()
	PopTransform()
	PushClipRect(Rect)
	PopClip()
}

// ShapeCanvas contains solid geometry operations. Implementations preserve
// painter order, the package's upper-left coordinate system, and premultiplied
// color semantics.
type ShapeCanvas interface {
	Clear(Color)
	DrawPoint(Point, Color)
	DrawLine(a, b Point, width Scalar, color Color)
	DrawPolyline(points []Point, width Scalar, color Color)
	FillRect(Rect, Color)
	StrokeRect(Rect, Scalar, Color)
	FillTriangle(a, b, c Point, color Color)
	FillPolygon(points []Point, rule FillRule, color Color)
	FillConvexPolygon(points []Point, color Color)
	FillEllipse(Rect, Color)
	StrokeEllipse(Rect, Scalar, Color)
	FillPath(*Path, FillRule, Color)
	StrokePath(*Path, Scalar, Color)
}

// ImageCanvas draws portable RGBA8 and A8 image resources.
type ImageCanvas interface {
	DrawImage(image *Image, src, dst Rect, sampling Sampling, tint Color)
}

// TextCanvas draws built-in, TrueType, and caller-positioned glyph text.
type TextCanvas interface {
	DrawText(font *Font, baseline Point, text string, color Color)
	DrawTextBytes(font *Font, baseline Point, text []byte, color Color)
	DrawGlyphRun(origin Point, glyphs []Glyph, color Color)
}

// DamageCanvas identifies independently repainted regions. It lets retained
// views preserve disjoint damage through either software or GPU presentation.
type DamageCanvas interface {
	BeginDamage(Rect)
	EndDamage()
}

// Canvas is the drawing contract accepted by Forms paint handlers. Surface is
// the concrete software implementation; accelerated window frames may provide
// another implementation without pretending to expose writable CPU pixels.
type Canvas interface {
	CanvasState
	ShapeCanvas
	ImageCanvas
	TextCanvas
	DamageCanvas
}

var _ Canvas = (*Surface)(nil)
