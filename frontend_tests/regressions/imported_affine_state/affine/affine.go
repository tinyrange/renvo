package affine

type Matrix struct {
	XX float64
	YX float64
	XY float64
	YY float64
	X0 float64
	Y0 float64
	W  float64
}

type Surface struct {
	Pixels    []byte
	Stride    int
	transform Matrix
}

func Identity() Matrix {
	return Matrix{XX: 1, YY: 1, W: 1}
}

func NewSurface(width, height int) *Surface {
	return &Surface{Pixels: make([]byte, width*height*4), Stride: width * 4, transform: Identity()}
}

func (surface *Surface) ResetTransform() {
	surface.transform = Identity()
}

func (surface *Surface) SetTranslation(x, y float64) {
	surface.transform = Matrix{XX: 1, YY: 1, X0: x, Y0: y, W: 1}
}

func (surface *Surface) SetTransform(transform *Matrix) {
	surface.transform = *transform
	surface.transform.W = 1
}

func (surface *Surface) SetLinear(xx, yx, xy, yy float64) {
	surface.transform.XX = xx
	surface.transform.YX = yx
	surface.transform.XY = xy
	surface.transform.YY = yy
	surface.transform.W = 1
}

func (surface *Surface) FillMarker() {
	if int(surface.transform.XX) == 1 && int(surface.transform.YY) == 1 && int(surface.transform.W) == 1 {
		surface.Pixels[3] = 255
	}
}
