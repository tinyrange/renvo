package resource

type Surface struct {
	Value int
}

type Image = Surface

func NewImage() *Image {
	return &Surface{Value: 1}
}

func (surface *Surface) Destroy() {
	surface.Value = 0
}

// A later use of the alias must resolve to the same canonical receiver type.
type Holder struct {
	Image *Image
}
