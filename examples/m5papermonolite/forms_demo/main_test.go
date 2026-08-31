package main

import (
	"testing"

	"renvo.dev/device/display/ssd1677"
	"renvo.dev/examples/device/fontcache"
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

func TestShowcaseBuildsAllPagesWithTrueTypeDerivedFonts(t *testing.T) {
	body := fontcache.TitleSubset(fontcache.PaperMonoFormsGlyphs)
	title := body
	if body == nil || title == nil || graphics.MeasureText(title, "PaperMono").Width <= 0 {
		t.Fatal("cached TrueType-derived fonts failed to load")
	}
	for page := pageInputs; page <= pageMore; page++ {
		var demo showcase
		demo.initialize(body, title, page)
		if demo.tabs.SelectedIndex() != page {
			t.Fatalf("selected tab = %d, want %d", demo.tabs.SelectedIndex(), page)
		}
		pages := [][]*forms.Control{demo.inputPage, demo.listPage, demo.motionPage, demo.morePage}
		for candidate, controls := range pages {
			if candidate == page && len(controls) == 0 || candidate != page && len(controls) != 0 {
				t.Fatalf("page %d retained %d controls while page %d is active", candidate, len(controls), page)
			}
		}
		if page == pageInputs && (demo.text == nil || demo.area == nil || demo.check == nil || demo.radioA == nil || demo.progress == nil) {
			t.Fatal("inputs page omitted an interactive control family")
		}
		if page == pageLists && (demo.combo == nil || demo.list == nil) {
			t.Fatal("lists page omitted an interactive control family")
		}
		if page == pageMotion && (demo.slider == nil || demo.number == nil || demo.split == nil) {
			t.Fatal("motion page omitted an interactive control family")
		}
	}
}

func TestShowcasePaintsDirectlyIntoRotatedMonochromeFrame(t *testing.T) {
	var pixels [ssd1677.FrameSize]byte
	for index := range pixels {
		pixels[index] = 0xff
	}
	surface := graphics.NewSurfaceBufferFormatPreserve(ssd1677.Width, ssd1677.Height, graphics.PixelMono1, pixels[:])
	surface.SetAffine(0, -1, 1, 0, 0, ssd1677.Height)
	var demo showcase
	font := fontcache.TitleSubset(fontcache.PaperMonoFormsGlyphs)
	demo.initialize(font, font, pageInputs)
	if !demo.form.Paint(surface) {
		t.Fatal("initial form paint reported no work")
	}
	black, white := 0, 0
	for _, value := range pixels {
		if value == 0 {
			black++
		}
		if value == 0xff {
			white++
		}
	}
	if black == 0 || white == 0 {
		t.Fatalf("monochrome paint bytes black=%d white=%d", black, white)
	}
}
