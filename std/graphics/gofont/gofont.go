// Package gofont provides embedded proportional and monospace TrueType fonts.
// It lets applications render the same antialiased text on every target
// without depending on fonts installed by the host operating system.
package gofont

import (
	"renvo.dev/std/graphics"
)

func New(pixelHeight graphics.Scalar) *graphics.Font {
	return newFont(regularData(), pixelHeight)
}

func NewMono(pixelHeight graphics.Scalar) *graphics.Font {
	return newFont(monoData(), pixelHeight)
}

func newFont(data []byte, pixelHeight graphics.Scalar) *graphics.Font {
	font, err := graphics.NewTrueTypeFont(data, pixelHeight)
	if err != nil {
		return nil
	}
	return font
}
