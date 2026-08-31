package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/display/ssd1677"
	"renvo.dev/examples/device/fontcache"
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
	"renvo.dev/std/strconv"
)

const (
	screenWidth      = 480
	screenHeight     = 800
	buttonDebounceMS = 20
	frontLight       = 160
	readerLineHeight = 38
)

var frame ssd1677.Monochrome

type readerPage struct {
	title string
	lines []string
}

var bookPages = []readerPage{
	{
		title: "a small machine",
		lines: []string{
			"renvo began with a small idea.",
			"a practical program should travel",
			"without a heavy toolchain.",
			"",
			"the compiler reads a useful part of go",
			"and carries it to many machines.",
			"the same source can meet a desktop,",
			"a tiny board, or a web page.",
			"",
			"small parts are easier to understand.",
			"clear limits make them safer to change.",
			"each new target tests the whole design.",
		},
	},
	{
		title: "paper and light",
		lines: []string{
			"this screen keeps an image without power.",
			"ink moves only when a page must change.",
			"",
			"a full update gives the panel a clean",
			"starting point. later pages use a fast",
			"differential update with a safe limit.",
			"",
			"the front light shines across the panel",
			"rather than through it. text can remain",
			"comfortable in a dark room.",
			"",
			"press the lower side button to continue.",
		},
	},
	{
		title: "turning pages",
		lines: []string{
			"a button press is a small event.",
			"the reader remembers it while the panel",
			"is busy moving ink.",
			"",
			"when the display is ready, forms clears",
			"the old text and paints the next page.",
			"only invalid controls need to be drawn.",
			"",
			"the panel still receives a complete packed",
			"plane. its fast waveform makes that transfer",
			"feel like a page turn rather than a reboot.",
		},
	},
	{
		title: "forms in motion",
		lines: []string{
			"forms owns the retained controls on screen.",
			"labels hold the title, prose, and page count.",
			"changing their text invalidates their bounds.",
			"",
			"the drawing surface maps portrait positions",
			"onto the native rotated display memory.",
			"black and white remain packed one bit each.",
			"",
			"a true type derived cache keeps the letters",
			"sharp without storing a second large font.",
			"the result is simple, but it is a real ui.",
		},
	},
	{
		title: "the last page",
		lines: []string{
			"this is the end of the small sample book.",
			"the upper side button can take you back.",
			"",
			"a later reader could load text from storage,",
			"wrap prose to the font, remember a",
			"place, and offer a library or settings page.",
			"",
			"the basic loop is already here:",
			"read input, update forms, paint damage,",
			"and ask the paper display to present it.",
			"",
			"small pieces can still make useful things.",
		},
	},
}

type buttonDebouncer struct {
	initialized bool
	raw         bool
	stable      bool
	changedAt   uint32
}

func (button *buttonDebouncer) update(pressed bool, now uint32) bool {
	if !button.initialized {
		button.initialized = true
		button.raw = pressed
		button.stable = pressed
		button.changedAt = now
		return false
	}
	if pressed != button.raw {
		button.raw = pressed
		button.changedAt = now
		return false
	}
	if button.stable != button.raw && now-button.changedAt >= buttonDebounceMS {
		button.stable = button.raw
		return button.stable
	}
	return false
}

type reader struct {
	form        forms.Form
	title       *forms.Label
	chapter     *forms.Label
	body        *pageView
	status      *forms.StatusBar
	page        int
	pendingPage int
	up, down    buttonDebouncer
}

type pageView struct {
	forms.Control
	font  *graphics.Font
	lines []string
}

func newPageView(font *graphics.Font) *pageView {
	view := &pageView{font: font}
	view.Control = *forms.NewControl()
	view.SetTabStop(false)
	view.SetAccessibilityRole(forms.AccessibilityRoleLabel)
	view.Paint = view.paint
	return view
}

func (view *pageView) setLines(lines []string) {
	view.lines = lines
	view.Invalidate()
}

func (view *pageView) paint(surface graphics.Canvas) {
	bounds := view.Bounds()
	for index, line := range view.lines {
		baseline := bounds.MinY + graphics.Scalar(index)*readerLineHeight + view.font.Metrics.Ascent
		surface.DrawText(view.font, graphics.Point{X: bounds.MinX, Y: baseline}, line, graphics.Black)
	}
}

func readerTheme() forms.Theme {
	return forms.Theme{
		Window:        graphics.White,
		Surface:       graphics.White,
		SurfaceRaised: graphics.White,
		Field:         graphics.White,
		Text:          graphics.Black,
		MutedText:     graphics.Black,
		Border:        graphics.Black,
		Hover:         graphics.Black,
		Selection:     graphics.Black,
		SelectionText: graphics.White,
		Accent:        graphics.Black,
		AccentText:    graphics.White,
		Disabled:      graphics.Black,
	}
}

func (reader *reader) initialize(font *graphics.Font) {
	reader.page = -1
	reader.form.Initialize(screenWidth, screenHeight)
	reader.form.ApplyTheme(readerTheme())

	reader.title = forms.NewLabel()
	reader.title.SetBounds(graphics.R(16, 18, 448, 38))
	reader.title.SetFont(font)
	reader.title.SetText("RENVO READER")
	reader.form.Add(&reader.title.Control)

	rule := forms.NewPanel()
	rule.SetBounds(graphics.R(16, 68, 448, 1))
	rule.SetBackground(graphics.Black)
	reader.form.Add(&rule.Control)

	reader.chapter = forms.NewLabel()
	reader.chapter.SetBounds(graphics.R(16, 82, 448, 38))
	reader.chapter.SetFont(font)
	reader.form.Add(&reader.chapter.Control)

	reader.body = newPageView(font)
	reader.body.SetBounds(graphics.R(16, 130, 448, 570))
	reader.form.Add(&reader.body.Control)

	reader.status = forms.NewStatusBar()
	reader.status.SetBounds(graphics.R(0, 750, screenWidth, 50))
	reader.status.SetFont(font)
	reader.form.Add(&reader.status.Control)
	reader.setPage(0)
}

func (reader *reader) setPage(page int) bool {
	if page < 0 {
		page = 0
	}
	if page >= len(bookPages) {
		page = len(bookPages) - 1
	}
	if page == reader.page {
		return false
	}
	reader.page = page
	contents := bookPages[page]
	reader.chapter.SetText(contents.title)
	reader.body.setLines(contents.lines)
	reader.status.SetText(pageStatus(page))
	print("PAPERMONO READER PAGE ", page+1, " / ", len(bookPages), "\n")
	return true
}

func pageStatus(page int) string {
	return "upper: back   lower: next   " +
		strconv.Itoa(page+1) + " / " + strconv.Itoa(len(bookPages))
}

func (reader *reader) queuePage(delta int) {
	target := reader.page + reader.pendingPage + delta
	if target < 0 {
		target = 0
	}
	if target >= len(bookPages) {
		target = len(bookPages) - 1
	}
	reader.pendingPage = target - reader.page
}

func (reader *reader) sampleButtonLevels(up, down bool, now uint32) {
	if reader.up.update(up, now) {
		reader.queuePage(-1)
	}
	if reader.down.update(down, now) {
		reader.queuePage(1)
	}
}

func (reader *reader) sampleButtons() {
	reader.sampleButtonLevels(board.ButtonA.Pressed(), board.ButtonB.Pressed(), board.Clock.Milliseconds())
}

func (reader *reader) applyPendingPage() bool {
	if reader.pendingPage == 0 {
		return false
	}
	target := reader.page + reader.pendingPage
	reader.pendingPage = 0
	return reader.setPage(target)
}

func (reader *reader) PollDuringRefresh() error {
	reader.sampleButtons()
	return nil
}

func fail(message string) {
	print(message)
	_ = board.Display.Shutdown()
	for {
	}
}

func halt(message string) {
	print(message)
	for {
	}
}

func main() {
	font := fontcache.TitleSubset(fontcache.PaperMonoFormsGlyphs)
	if font == nil {
		halt("RENVO PAPERMONO-LITE READER FONT FAIL\n")
	}
	if err := board.Initialize(); err != nil {
		halt("RENVO PAPERMONO-LITE READER BOARD FAIL\n")
	}
	if err := board.Display.Enable(); err != nil {
		fail("RENVO PAPERMONO-LITE READER POWER FAIL\n")
	}
	if err := board.Power.SetFrontLight(frontLight); err != nil {
		fail("RENVO PAPERMONO-LITE READER FRONTLIGHT FAIL\n")
	}

	frame.Fill(true)
	surface := graphics.NewSurfaceBufferFormatPreserve(
		ssd1677.Width, ssd1677.Height, graphics.PixelMono1, frame[:])
	if surface == nil {
		fail("RENVO PAPERMONO-LITE READER SURFACE FAIL\n")
	}
	surface.SetAffine(0, -1, 1, 0, 0, ssd1677.Height)

	var app reader
	app.initialize(font)
	app.sampleButtons()
	if !app.form.Paint(surface) {
		fail("RENVO PAPERMONO-LITE READER PAINT FAIL\n")
	}
	if err := board.Display.FullMonochrome(frame[:]); err != nil {
		fail("RENVO PAPERMONO-LITE READER DISPLAY FAIL\n")
	}
	surface.ResetDirty()
	print("RENVO PAPERMONO-LITE READER READY\n")

	for {
		app.sampleButtons()
		app.applyPendingPage()
		if app.form.Paint(surface) {
			if err := board.Display.FastMonochrome(frame[:], &app); err != nil {
				fail("RENVO PAPERMONO-LITE READER PRESENT FAIL\n")
			}
			surface.ResetDirty()
		}
		board.Clock.DelayMilliseconds(1)
	}
}
