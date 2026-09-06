//go:build ignore

// Command font_cache_generate rasterizes the repository's Go Regular TrueType
// font on the host. Device examples consume the resulting A8 glyph masks
// directly, avoiding outline parsing and rasterization on the target CPU.
package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"os"

	"renvo.dev/examples/device/fontcache"
	"renvo.dev/std/graphics"
	"renvo.dev/std/graphics/gofont"
)

const firstGlyph = 32
const lastGlyph = 126

func appendFixed(buffer *bytes.Buffer, value graphics.Scalar) {
	binary.Write(buffer, binary.LittleEndian, int32(math.Round(float64(value)*65536)))
}

func includes(characters string, codepoint int) bool {
	if characters == "" {
		return true
	}
	for index := 0; index < len(characters); index++ {
		if int(characters[index]) == codepoint {
			return true
		}
	}
	return false
}

func generate(pixelHeight graphics.Scalar, output, characters string) {
	font := gofont.New(pixelHeight)
	if font == nil {
		panic("failed to load Go Regular")
	}
	var buffer bytes.Buffer
	buffer.WriteString("RGF1")
	appendFixed(&buffer, font.Metrics.Ascent)
	appendFixed(&buffer, font.Metrics.Descent)
	appendFixed(&buffer, font.Metrics.LineGap)
	count := 0
	for codepoint := firstGlyph; codepoint <= lastGlyph; codepoint++ {
		if includes(characters, codepoint) {
			count++
		}
	}
	binary.Write(&buffer, binary.LittleEndian, uint16(count))

	const canvasSize = 64
	const originX = 8
	baseline := graphics.Scalar(8) + font.Metrics.Ascent
	for codepoint := firstGlyph; codepoint <= lastGlyph; codepoint++ {
		if !includes(characters, codepoint) {
			continue
		}
		canvas := graphics.NewImageFormat(canvasSize, canvasSize, graphics.PixelA8, nil)
		canvas.DrawText(font, graphics.Point{X: originX, Y: baseline}, string(rune(codepoint)), graphics.RGBA(255, 255, 255, 255))
		minX, minY, maxX, maxY := canvasSize, canvasSize, 0, 0
		for y := 0; y < canvasSize; y++ {
			for x := 0; x < canvasSize; x++ {
				if canvas.Pixels[y*canvas.Stride+x] == 0 {
					continue
				}
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x+1 > maxX {
					maxX = x + 1
				}
				if y+1 > maxY {
					maxY = y + 1
				}
			}
		}
		if maxX == 0 || maxY == 0 {
			minX, minY = originX, int(baseline)
			maxX, maxY = minX, minY
		}
		width, height := maxX-minX, maxY-minY
		binary.Write(&buffer, binary.LittleEndian, uint32(codepoint))
		appendFixed(&buffer, graphics.Scalar(minX)-originX)
		appendFixed(&buffer, graphics.Scalar(minY)-baseline)
		appendFixed(&buffer, graphics.MeasureText(font, string(rune(codepoint))).Width)
		binary.Write(&buffer, binary.LittleEndian, uint16(width))
		binary.Write(&buffer, binary.LittleEndian, uint16(height))
		for y := minY; y < maxY; y++ {
			for x := minX; x < maxX; x++ {
				buffer.WriteByte(canvas.Pixels[y*canvas.Stride+x])
			}
		}
	}
	if err := os.WriteFile(output, buffer.Bytes(), 0o644); err != nil {
		panic(err)
	}
}

func main() {
	generate(18, "Go-Regular-18.rgf", "")
	generate(26, "Go-Regular-26.rgf", "")
	generate(26, "Go-Regular-26-PaperMono-Forms.rgf", fontcache.PaperMonoFormsGlyphs)
}
