// Package gofont provides an embedded proportional TrueType UI font. It lets
// Forms applications render the same antialiased text on every target without
// depending on fonts installed by the host operating system.
package gofont

import (
	"renvo.dev/std/graphics"
)

func New(pixelHeight graphics.Scalar) *graphics.Font {
	data := regularData()
	font, err := graphics.NewTrueTypeFont(data, pixelHeight)
	if err != nil {
		return nil
	}
	return font
}
