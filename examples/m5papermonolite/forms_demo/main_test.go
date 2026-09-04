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

func touch(demo *showcase, x, y int) {
	demo.samplePointer(x, y, true)
	demo.dispatchCaptured()
	demo.samplePointer(x, y, false)
	demo.dispatchCaptured()
}

func TestTouchBridgeRoutesTapsAndDragsToFormsControls(t *testing.T) {
	font := fontcache.TitleSubset(fontcache.PaperMonoFormsGlyphs)
	if font == nil {
		t.Fatal("cached font failed to load")
	}

	var inputs showcase
	inputs.initialize(font, font, pageInputs)
	touch(&inputs, 80, 378)
	if inputs.check.Checked() {
		t.Fatal("touch did not toggle the checkbox")
	}
	touch(&inputs, 200, 504)
	if inputs.progress.Value() != 45 {
		t.Fatalf("button touch set progress to %d, want 45", inputs.progress.Value())
	}
	touch(&inputs, 300, 112)
	if inputs.requestedPage != pageMotion {
		t.Fatalf("tab touch requested page %d, want %d", inputs.requestedPage, pageMotion)
	}

	var lists showcase
	lists.initialize(font, font, pageLists)
	touch(&lists, 80, 266)
	if lists.list.SelectedIndex() != 1 {
		t.Fatalf("list touch selected row %d, want 1", lists.list.SelectedIndex())
	}

	var motion showcase
	motion.initialize(font, font, pageMotion)
	motion.samplePointer(180, 212, true)
	motion.dispatchCaptured()
	if !motion.touchActive() {
		t.Fatal("touch bridge did not retain the active drag")
	}
	motion.samplePointer(410, 212, true)
	motion.dispatchCaptured()
	motion.samplePointer(410, 212, false)
	motion.dispatchCaptured()
	if motion.slider.Value() < 80 {
		t.Fatalf("slider drag produced %d, want at least 80", motion.slider.Value())
	}
	beforeStep := motion.number.Value()
	touch(&motion, 450, 294)
	if motion.number.Value() != beforeStep+5 || motion.slider.Value() != motion.number.Value() {
		t.Fatalf("stepper touch produced number=%d slider=%d from %d", motion.number.Value(), motion.slider.Value(), beforeStep)
	}
	motion.samplePointer(238, 500, true)
	motion.dispatchCaptured()
	motion.samplePointer(310, 500, true)
	motion.dispatchCaptured()
	motion.samplePointer(310, 500, false)
	motion.dispatchCaptured()
	if motion.split.SplitterDistance() < 280 {
		t.Fatalf("splitter drag produced distance %d, want at least 280", motion.split.SplitterDistance())
	}
	if motion.dispatchedDown || motion.eventCount != 0 {
		t.Fatal("touch bridge retained a pressed or queued event after release")
	}
}

func TestTouchQueuePreservesReleaseWhenMoveSamplesOverflow(t *testing.T) {
	font := fontcache.TitleSubset(fontcache.PaperMonoFormsGlyphs)
	var demo showcase
	demo.initialize(font, font, pageMotion)
	demo.samplePointer(40, 212, true)
	for x := 41; x < 200; x++ {
		demo.samplePointer(x, 212, true)
	}
	demo.samplePointer(200, 212, false)
	if demo.eventCount != maximumQueuedPointers {
		t.Fatalf("queued events = %d, want %d", demo.eventCount, maximumQueuedPointers)
	}
	demo.dispatchCaptured()
	if demo.dispatchedDown || demo.sampledPressed || demo.eventCount != 0 {
		t.Fatal("overflowed touch queue lost its final release")
	}
}
