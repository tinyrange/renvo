package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"renvo.dev/device/display/ssd1677"
	"renvo.dev/examples/device/fontcache"
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

func renderShowcasePreview(t *testing.T) *image.Gray {
	t.Helper()
	var pixels [ssd1677.FrameSize]byte
	for index := range pixels {
		pixels[index] = 0xff
	}
	surface := graphics.NewSurfaceBufferFormatPreserve(ssd1677.Width, ssd1677.Height, graphics.PixelMono1, pixels[:])
	surface.SetAffine(0, -1, 1, 0, 0, ssd1677.Height)
	font := fontcache.TitleSubset(fontcache.PaperMonoFormsGlyphs)
	var demo showcase
	demo.initialize(font, font, pageInputs)
	if !demo.form.Paint(surface) {
		t.Fatal("initial preview paint reported no work")
	}
	preview := image.NewGray(image.Rect(0, 0, screenWidth, screenHeight))
	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
			nativeX, nativeY := y, screenWidth-1-x
			mask := byte(0x80 >> uint(nativeX&7))
			value := uint8(0)
			if pixels[nativeY*ssd1677.BytesPerRow+nativeX/8]&mask != 0 {
				value = 255
			}
			preview.SetGray(x, y, color.Gray{Y: value})
		}
	}
	return preview
}

func TestShowcaseMatchesHostPreview(t *testing.T) {
	preview := renderShowcasePreview(t)
	path := filepath.Join("testdata", "forms-demo.png")
	if os.Getenv("RENVO_UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = png.Encode(file, preview); err != nil {
			file.Close()
			t.Fatal(err)
		}
		if err = file.Close(); err != nil {
			t.Fatal(err)
		}
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := png.Decode(file)
	file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if want.Bounds() != preview.Bounds() {
		t.Fatalf("preview bounds = %v, want %v", preview.Bounds(), want.Bounds())
	}
	for y := 0; y < screenHeight; y++ {
		for x := 0; x < screenWidth; x++ {
			gotGray := color.GrayModel.Convert(preview.At(x, y)).(color.Gray)
			wantGray := color.GrayModel.Convert(want.At(x, y)).(color.Gray)
			if gotGray != wantGray {
				t.Fatalf("preview pixel %d,%d = %d, want %d", x, y, gotGray.Y, wantGray.Y)
			}
		}
	}
}

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
