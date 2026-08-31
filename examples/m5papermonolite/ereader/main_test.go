package main

import (
	"testing"

	"renvo.dev/device/display/ssd1677"
	"renvo.dev/examples/device/fontcache"
	"renvo.dev/std/graphics"
)

func TestButtonDebouncerReportsOnlyStablePresses(t *testing.T) {
	var button buttonDebouncer
	if button.update(false, 0) || button.update(true, 2) || button.update(false, 5) || button.update(true, 8) {
		t.Fatal("button bounce produced a press")
	}
	if button.update(true, 27) {
		t.Fatal("button became stable before the debounce interval")
	}
	if !button.update(true, 28) {
		t.Fatal("stable button press was not reported")
	}
	if button.update(true, 100) || button.update(false, 101) || button.update(false, 121) {
		t.Fatal("held button or release produced a press")
	}
	if button.update(true, 130) || !button.update(true, 150) {
		t.Fatal("second stable press was not reported exactly once")
	}
}

func TestReaderPageQueueClampsAndCoalesces(t *testing.T) {
	font := fontcache.TitleSubset(fontcache.PaperMonoFormsGlyphs)
	var app reader
	app.initialize(font)
	app.queuePage(-1)
	if app.pendingPage != 0 {
		t.Fatalf("previous from first page queued %d", app.pendingPage)
	}
	app.queuePage(1)
	app.queuePage(1)
	if app.pendingPage != 2 || !app.applyPendingPage() || app.page != 2 {
		t.Fatalf("two next presses produced page=%d pending=%d", app.page, app.pendingPage)
	}
	for index := 0; index < 20; index++ {
		app.queuePage(1)
	}
	app.applyPendingPage()
	if app.page != len(bookPages)-1 {
		t.Fatalf("forward clamp produced page %d", app.page)
	}
	for index := 0; index < 20; index++ {
		app.queuePage(-1)
	}
	app.applyPendingPage()
	if app.page != 0 {
		t.Fatalf("backward clamp produced page %d", app.page)
	}
}

func TestReaderPagesFitAndPaintRotatedMonochrome(t *testing.T) {
	font := fontcache.TitleSubset(fontcache.PaperMonoFormsGlyphs)
	if font == nil {
		t.Fatal("reader font failed to load")
	}
	if width := graphics.MeasureText(font, pageStatus(len(bookPages)-1)).Width; width > screenWidth-12 {
		t.Fatalf("reader status width %v exceeds the screen", width)
	}
	assertText := func(name, text string) {
		for index := 0; index < len(text); index++ {
			available := false
			for _, glyph := range []byte(fontcache.PaperMonoFormsGlyphs) {
				if text[index] == glyph {
					available = true
					break
				}
			}
			if !available {
				t.Fatalf("%s uses uncached glyph %q", name, text[index])
			}
		}
	}
	assertText("reader title", "RENVO READER")
	assertText("reader status", pageStatus(len(bookPages)-1))
	for pageIndex, page := range bookPages {
		if graphics.Scalar(len(page.lines))*readerLineHeight > 570 {
			t.Fatalf("page %d has %d lines and exceeds the body height", pageIndex, len(page.lines))
		}
		for lineIndex, line := range append([]string{page.title}, page.lines...) {
			assertText("reader page text", line)
			if width := graphics.MeasureText(font, line).Width; width > 448 {
				t.Fatalf("page %d line %d width %v exceeds 448: %q", pageIndex, lineIndex, width, line)
			}
		}
	}

	var pixels [ssd1677.FrameSize]byte
	for index := range pixels {
		pixels[index] = 0xff
	}
	surface := graphics.NewSurfaceBufferFormatPreserve(
		ssd1677.Width, ssd1677.Height, graphics.PixelMono1, pixels[:])
	surface.SetAffine(0, -1, 1, 0, 0, ssd1677.Height)
	var app reader
	app.initialize(font)
	for page := 0; page < len(bookPages); page++ {
		app.setPage(page)
		if !app.form.Paint(surface) {
			t.Fatalf("page %d did not paint", page)
		}
	}
	black := 0
	for _, value := range pixels {
		if value != 0xff {
			black++
		}
	}
	if black == 0 {
		t.Fatal("reader pages left the monochrome framebuffer blank")
	}
}
