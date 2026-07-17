//go:build rtg && !darwin

package widget

func NewWindow(options Options) *Window {
	return newWindow(options)
}
