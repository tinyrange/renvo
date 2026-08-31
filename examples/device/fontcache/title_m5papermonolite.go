//go:build m5papermonolite

package fontcache

import (
	_ "embed"

	"renvo.dev/std/graphics"
)

//go:embed Go-Regular-26-PaperMono-Forms.rgf
var titleCache string

var titleFont *graphics.Font

// Title returns the PaperMono Forms showcase's 26-pixel cached glyph subset.
func Title() *graphics.Font {
	if titleFont == nil {
		titleFont = load(titleCache, "")
	}
	return titleFont
}

// TitleSubset loads only requested descriptors from the PaperMono Forms glyph
// set. Glyph A8 masks continue to alias embedded data rather than consuming
// the PaperMono-Lite's managed arena.
func TitleSubset(characters string) *graphics.Font { return load(titleCache, characters) }

// Body shares the 26-pixel PaperMono cache so the constrained build embeds a
// single glyph data set.
func Body() *graphics.Font { return Title() }

// BodySubset is the subset-loading counterpart to Body.
func BodySubset(characters string) *graphics.Font { return TitleSubset(characters) }
