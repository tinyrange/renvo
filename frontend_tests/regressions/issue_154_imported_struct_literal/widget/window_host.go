//go:build !renvo

package widget

func NewWindow(options Options) *Window {
	return newWindow(options)
}
