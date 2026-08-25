// Package fontcache provides the cached Go Regular fonts used by Tab5 demos.
package fontcache

import (
	_ "embed"

	"renvo.dev/std/graphics"
)

// These caches are generated from std/graphics/gofont/Go-Regular.ttf and
// remain covered by std/graphics/gofont/LICENSE.
//
//go:generate go run ../forms_demo/font_cache_generate.go

//go:embed Go-Regular-18.rgf
var bodyCache string

//go:embed Go-Regular-26.rgf
var titleCache string

func uint16At(data string, at int) int {
	return int(data[at]) | int(data[at+1])<<8
}

func uint32At(data string, at int) uint32 {
	return uint32(data[at]) | uint32(data[at+1])<<8 |
		uint32(data[at+2])<<16 | uint32(data[at+3])<<24
}

func scalarAt(data string, at int) graphics.Scalar {
	return graphics.Scalar(int32(uint32At(data, at))) / 65536
}

func load(data string) *graphics.Font {
	if len(data) < 18 || data[:4] != "RGF1" {
		return nil
	}
	metrics := graphics.FontMetrics{
		Ascent:  scalarAt(data, 4),
		Descent: scalarAt(data, 8),
		LineGap: scalarAt(data, 12),
	}
	count, at := uint16At(data, 16), 18
	glyphs := make([]graphics.RasterGlyph, count)
	for index := 0; index < count; index++ {
		if at+20 > len(data) {
			return nil
		}
		codepoint := int(uint32At(data, at))
		xOffset, yOffset := scalarAt(data, at+4), scalarAt(data, at+8)
		advance := scalarAt(data, at+12)
		width, height := uint16At(data, at+16), uint16At(data, at+18)
		at += 20
		size := width * height
		if size < 0 || at+size > len(data) {
			return nil
		}
		var mask *graphics.Image
		if size > 0 {
			pixels := make([]byte, size)
			for pixel := range pixels {
				pixels[pixel] = data[at+pixel]
			}
			mask = graphics.NewSurfaceBufferFormatPreserve(width, height, graphics.PixelA8, pixels)
		}
		glyphs[index] = graphics.RasterGlyph{
			Codepoint: codepoint, Mask: mask, XOffset: xOffset,
			YOffset: yOffset, Advance: advance,
		}
		at += size
	}
	if at != len(data) {
		return nil
	}
	return graphics.NewRasterFont(metrics, glyphs)
}

// Body returns the 18-pixel cached font.
func Body() *graphics.Font { return load(bodyCache) }

// Title returns the 26-pixel cached font.
func Title() *graphics.Font { return load(titleCache) }
