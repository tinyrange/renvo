package widget

type Options struct {
	Title  string
	Width  int
	Height int
	Hidden bool
}

type Window struct {
	area int
}

func (window *Window) Area() int {
	return window.area
}
