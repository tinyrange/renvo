//go:build rtg && darwin && arm64

package widget

func NewWindow(options Options) *Window {
	return newWindow(options)
}
