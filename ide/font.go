package ide

import (
	"renvo.dev/std/graphics"
	"renvo.dev/std/graphics/gofont"
	renvoos "renvo.dev/std/os"
)

// NewUIFont returns a compact monospace font suitable for both explorer and
// editor controls. Native installations use a platform font when one is
// available; unusual and embedded targets retain the allocation-free bitmap
// fallback.
func NewUIFont() *graphics.Font {
	paths := []string{
		"/System/Library/Fonts/SFNSMono.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
		"/usr/share/fonts/truetype/liberation2/LiberationMono-Regular.ttf",
		"C:/Windows/Fonts/consola.ttf",
		"C:/Windows/Fonts/lucon.ttf",
		"C:/Windows/Fonts/cour.ttf",
	}
	for i := 0; i < len(paths); i++ {
		data, readError := renvoos.ReadFile(paths[i])
		if readError != nil {
			continue
		}
		font, fontError := graphics.NewTrueTypeFont(data, 16)
		if fontError == nil {
			return font
		}
	}
	if font := gofont.NewMono(16); font != nil {
		return font
	}
	return graphics.NewBuiltinFont(2)
}

// NewInterfaceFont returns the proportional face used by application chrome,
// tree rows, labels, property sheets, and future Forms controls. The editor
// keeps NewUIFont's monospace contract.
func NewInterfaceFont() *graphics.Font {
	paths := []string{
		"/System/Library/Fonts/SFNS.ttf",
		"/System/Library/Fonts/Helvetica.ttc",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation2/LiberationSans-Regular.ttf",
		"C:/Windows/Fonts/segoeui.ttf",
		"C:/Windows/Fonts/tahoma.ttf",
		"C:/Windows/Fonts/arial.ttf",
	}
	for i := 0; i < len(paths); i++ {
		data, readError := renvoos.ReadFile(paths[i])
		if readError != nil {
			continue
		}
		font, fontError := graphics.NewTrueTypeFont(data, 15)
		if fontError == nil {
			return font
		}
	}
	if font := gofont.New(15); font != nil {
		return font
	}
	return graphics.NewBuiltinFont(2)
}

func fontPixelCeil(value graphics.Scalar) int {
	pixels := int(value)
	if graphics.Scalar(pixels) < value {
		pixels++
	}
	if pixels < 1 {
		pixels = 1
	}
	return pixels
}

func fontLineHeight(font *graphics.Font) int {
	if font == nil {
		return 1
	}
	return fontPixelCeil(font.Metrics.Ascent + font.Metrics.Descent + font.Metrics.LineGap)
}

func fontCellWidth(font *graphics.Font) graphics.Scalar {
	width := graphics.MeasureText(font, "M").Width
	if width < 1 {
		return 1
	}
	return width
}
