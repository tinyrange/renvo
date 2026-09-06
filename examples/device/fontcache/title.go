//go:build !m5papermonolite

package fontcache

import (
	_ "embed"

	"renvo.dev/std/graphics"
)

//go:embed Go-Regular-26.rgf
var titleCache string

var titleFont *graphics.Font

// Title returns the shared 26-pixel cached font.
func Title() *graphics.Font {
	if titleFont == nil {
		titleFont = load(&titleCache, "")
	}
	return titleFont
}

// TitleSubset loads the 26-pixel cache while retaining only the requested
// ASCII glyphs, plus the shared space and question-mark fallbacks.
func TitleSubset(characters string) *graphics.Font { return load(&titleCache, characters) }
