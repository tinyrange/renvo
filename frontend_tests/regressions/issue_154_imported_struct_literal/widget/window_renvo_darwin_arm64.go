//go:build renvo && darwin && arm64

package widget

func NewWindow(options Options) *Window {
	return newWindow(options)
}
