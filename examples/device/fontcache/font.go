// Package fontcache provides cached Go Regular fonts used by device demos.
package fontcache

import (
	"renvo.dev/std/graphics"
)

// PaperMonoFormsGlyphs is the complete printable set used by the constrained
// PaperMono-Lite Forms showcase cache.
const PaperMonoFormsGlyphs = " ,-./0123456789:;?ABCDEFGILMNOPRSTUVabcdefghiklmnopqrstuvwxy"

// These caches are generated from std/graphics/gofont/Go-Regular.ttf and
// remain covered by std/graphics/gofont/LICENSE.
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

func containsCodepoint(characters string, codepoint int) bool {
	for at := 0; at < len(characters); at++ {
		if int(characters[at]) == codepoint {
			return true
		}
	}
	return false
}

func load(data *string, characters string) *graphics.Font {
	if data == nil || len(*data) < 18 || (*data)[:4] != "RGF1" {
		return nil
	}
	source := *data
	metrics := graphics.FontMetrics{
		Ascent:  scalarAt(source, 4),
		Descent: scalarAt(source, 8),
		LineGap: scalarAt(source, 12),
	}
	count, at := uint16At(source, 16), 18
	capacity := count
	if characters != "" && len(characters) < capacity {
		capacity = len(characters)
	}
	font := graphics.NewRasterFontCapacity(metrics, capacity)
	for index := 0; index < count; index++ {
		if at+20 > len(source) {
			return nil
		}
		codepoint := int(uint32At(source, at))
		xOffset, yOffset := scalarAt(source, at+4), scalarAt(source, at+8)
		advance := scalarAt(source, at+12)
		width, height := uint16At(source, at+16), uint16At(source, at+18)
		at += 20
		size := width * height
		if size < 0 || at+size > len(source) {
			return nil
		}
		if characters == "" || codepoint == ' ' || codepoint == '?' || containsCodepoint(characters, codepoint) {
			font.AddRasterGlyph(graphics.RasterGlyph{
				Codepoint: codepoint, MaskSource: data, MaskOffset: at, MaskWidth: width,
				MaskHeight: height, MaskStride: width, XOffset: xOffset,
				YOffset: yOffset, Advance: advance,
			})
		}
		at += size
	}
	if at != len(source) {
		return nil
	}
	return font
}
