// Package graphics provides RTG's portable windowing and two-dimensional
// rendering API. Coordinates use an upper-left origin and half-open rectangles.
package graphics

type Scalar = float64

type Point struct {
	X Scalar
	Y Scalar
}

type Rect struct {
	MinX Scalar
	MinY Scalar
	MaxX Scalar
	MaxY Scalar
}

func R(x, y, width, height Scalar) Rect {
	return Rect{MinX: x, MinY: y, MaxX: x + width, MaxY: y + height}
}

func (r Rect) Width() Scalar  { return r.MaxX - r.MinX }
func (r Rect) Height() Scalar { return r.MaxY - r.MinY }
func (r Rect) Empty() bool    { return r.MaxX <= r.MinX || r.MaxY <= r.MinY }

type Color struct {
	R byte
	G byte
	B byte
	A byte
}

// RGBA constructs a premultiplied-alpha color from straight RGBA channels.
func RGBA(r, g, b, a byte) Color {
	alpha := int(a)
	return Color{
		R: byte((int(r)*alpha + 127) / 255),
		G: byte((int(g)*alpha + 127) / 255),
		B: byte((int(b)*alpha + 127) / 255),
		A: a,
	}
}

var Transparent = Color{}
var Black = Color{R: 0, G: 0, B: 0, A: 255}
var White = Color{R: 255, G: 255, B: 255, A: 255}

type BlendMode int

const (
	BlendCopy BlendMode = iota
	BlendSourceOver
)

type Sampling int

const (
	SamplingNearest Sampling = iota
	SamplingLinear
)

type PixelFormat int

const (
	PixelRGBA8 PixelFormat = iota
	PixelA8
)

// Mat2x3 is an affine matrix. TransformPoint computes
// (A*x + C*y + TX, B*x + D*y + TY).
type Mat2x3 struct {
	A  Scalar
	B  Scalar
	C  Scalar
	D  Scalar
	TX Scalar
	TY Scalar
}

func Identity() Mat2x3 { return Mat2x3{A: 1, B: 0, C: 0, D: 1, TX: 0, TY: 0} }

func Translate(x, y Scalar) Mat2x3 { return Mat2x3{A: 1, B: 0, C: 0, D: 1, TX: x, TY: y} }

func Scale(x, y Scalar) Mat2x3 { return Mat2x3{A: x, B: 0, C: 0, D: y, TX: 0, TY: 0} }

func (m *Mat2x3) TransformPoint(p Point) Point {
	return Point{X: m.A*p.X + m.C*p.Y + m.TX, Y: m.B*p.X + m.D*p.Y + m.TY}
}

func (m *Mat2x3) Mul(n *Mat2x3) Mat2x3 {
	return Mat2x3{
		A:  m.A*n.A + m.C*n.B,
		B:  m.B*n.A + m.D*n.B,
		C:  m.A*n.C + m.C*n.D,
		D:  m.B*n.C + m.D*n.D,
		TX: m.A*n.TX + m.C*n.TY + m.TX,
		TY: m.B*n.TX + m.D*n.TY + m.TY,
	}
}
