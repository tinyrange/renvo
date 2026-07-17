//go:build !rtg

package widget

func NewWindow(options Options) *Window {
	return newWindow(options)
}
