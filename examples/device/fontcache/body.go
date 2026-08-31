//go:build !m5papermonolite

package fontcache

import (
	_ "embed"

	"renvo.dev/std/graphics"
)

//go:generate go run ../forms_demo/font_cache_generate.go

//go:embed Go-Regular-18.rgf
var bodyCache string

var bodyFont *graphics.Font

// Body returns the shared 18-pixel cached font.
func Body() *graphics.Font {
	if bodyFont == nil {
		bodyFont = load(&bodyCache, "")
	}
	return bodyFont
}

// BodySubset loads the 18-pixel cache while retaining only the requested
// ASCII glyphs. A question-mark fallback and space metrics are always kept.
func BodySubset(characters string) *graphics.Font { return load(&bodyCache, characters) }
