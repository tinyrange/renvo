package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/display/ssd1677"
	"renvo.dev/examples/device/fontcache"
	"renvo.dev/forms"
	"renvo.dev/internal/arena"
	"renvo.dev/std/graphics"
)

const (
	screenWidth      = 480
	screenHeight     = 800
	buttonDebounceMS = 20
	settingsHoldMS   = 700
	frontLight       = 160
	readerLineHeight = 38
	maximumPointers  = 64
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

var pageStatuses = []string{
	"upper: back   lower: next   1 / 5",
	"upper: back   lower: next   2 / 5",
	"upper: back   lower: next   3 / 5",
	"upper: back   lower: next   4 / 5",
	"upper: back   lower: next   5 / 5",
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
	form              forms.Form
	title             *forms.Label
	chapter           *forms.Label
	body              *pageView
	status            *forms.StatusBar
	brightness        *forms.Slider
	red, green, blue  *forms.Slider
	font              *graphics.Font
	page              int
	pendingPage       int
	settings          bool
	requestedView     int
	brightnessValue   int
	redValue          int
	greenValue        int
	blueValue         int
	appliedBrightness int
	appliedRed        int
	appliedGreen      int
	appliedBlue       int
	up, down          buttonDebouncer
	upPressedAt       uint32
	upHeld            bool

	events                         [maximumPointers]pointerEvent
	eventCount                     int
	sampledX, sampledY             int
	sampledPressed, dispatchedDown bool
	dispatchedX, dispatchedY       int
}

type pointerEvent struct {
	x, y    int
	pressed bool
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

func newReaderLabel(font *graphics.Font, text string, bounds graphics.Rect) *forms.Label {
	label := forms.NewLabel()
	label.SetBounds(bounds)
	label.SetFont(font)
	label.SetText(text)
	return label
}

func (reader *reader) initialize(font *graphics.Font, page int) {
	reader.font = font
	reader.requestedView = -1
	reader.form.Initialize(screenWidth, screenHeight)
	reader.form.ApplyTheme(readerTheme())
	if reader.settings {
		reader.initializeSettings(font)
	} else {
		reader.initializePage(font, page)
	}
	reader.form.ReserveInvalidRects(16)
}

func (reader *reader) initializePage(font *graphics.Font, page int) {
	reader.title = newReaderLabel(font, "RENVO READER", graphics.R(16, 18, 448, 38))
	reader.form.Add(&reader.title.Control)

	rule := forms.NewPanel()
	rule.SetBounds(graphics.R(16, 68, 448, 1))
	rule.SetBackground(graphics.Black)
	reader.form.Add(&rule.Control)

	reader.chapter = newReaderLabel(font, "", graphics.R(16, 82, 448, 38))
	reader.form.Add(&reader.chapter.Control)

	reader.body = newPageView(font)
	reader.body.SetBounds(graphics.R(16, 130, 448, 570))
	reader.form.Add(&reader.body.Control)

	reader.status = forms.NewStatusBar()
	reader.status.SetBounds(graphics.R(0, 750, screenWidth, 50))
	reader.status.SetFont(font)
	reader.form.Add(&reader.status.Control)
	reader.page = -1
	reader.setPage(page)
}

func (reader *reader) initializeSettings(font *graphics.Font) {
	settingsTitle := newReaderLabel(font, "READER SETTINGS", graphics.R(16, 18, 448, 38))
	reader.form.Add(&settingsTitle.Control)
	settingsRule := forms.NewPanel()
	settingsRule.SetBounds(graphics.R(16, 68, 448, 1))
	settingsRule.SetBackground(graphics.Black)
	reader.form.Add(&settingsRule.Control)

	labels := []*forms.Label{
		newReaderLabel(font, "Front light", graphics.R(16, 110, 448, 38)),
		newReaderLabel(font, "Red - on / off", graphics.R(16, 245, 448, 38)),
		newReaderLabel(font, "Green", graphics.R(16, 380, 448, 38)),
		newReaderLabel(font, "Blue", graphics.R(16, 515, 448, 38)),
	}
	for _, label := range labels {
		reader.form.Add(&label.Control)
	}

	reader.brightness = forms.NewSlider()
	reader.red = forms.NewSlider()
	reader.green = forms.NewSlider()
	reader.blue = forms.NewSlider()
	sliders := []*forms.Slider{reader.brightness, reader.red, reader.green, reader.blue}
	positions := []graphics.Scalar{155, 290, 425, 560}
	for index, slider := range sliders {
		slider.SetBounds(graphics.R(16, positions[index], 448, 64))
		slider.SetRange(0, 255)
		reader.form.Add(&slider.Control)
	}
	reader.brightness.SetValue(reader.brightnessValue)
	reader.red.SetValue(reader.redValue)
	reader.green.SetValue(reader.greenValue)
	reader.blue.SetValue(reader.blueValue)

	settingsStatus := forms.NewStatusBar()
	settingsStatus.SetBounds(graphics.R(0, 750, screenWidth, 50))
	settingsStatus.SetFont(font)
	settingsStatus.SetText("hold upper: close   drag to change")
	reader.form.Add(&settingsStatus.Control)
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
	if page < 0 {
		page = 0
	}
	if page >= len(pageStatuses) {
		page = len(pageStatuses) - 1
	}
	return pageStatuses[page]
}

func (reader *reader) showSettings(show bool) {
	requested := 0
	if show {
		requested = 1
	}
	if reader.settings == show || reader.requestedView == requested {
		return
	}
	reader.requestedView = requested
}

func (reader *reader) syncSettingsValues() {
	if !reader.settings || reader.brightness == nil {
		return
	}
	reader.brightnessValue = reader.brightness.Value()
	reader.redValue = reader.red.Value()
	reader.greenValue = reader.green.Value()
	reader.blueValue = reader.blue.Value()
}

func (r *reader) rebuildRequestedView(lowMark, highMark int) bool {
	if r.requestedView < 0 || r.touchActive() {
		return false
	}
	r.syncSettingsValues()
	show := r.requestedView == 1
	font, page, pendingPage := r.font, r.page, r.pendingPage
	brightness, red := r.brightnessValue, r.redValue
	green, blue := r.greenValue, r.blueValue
	appliedBrightness, appliedRed := r.appliedBrightness, r.appliedRed
	appliedGreen, appliedBlue := r.appliedGreen, r.appliedBlue
	up, down := r.up, r.down
	upPressedAt, upHeld := r.upPressedAt, r.upHeld
	arena.Reset(lowMark)
	arena.PersistReset(highMark)
	*r = reader{
		settings: show, page: page, pendingPage: pendingPage,
		brightnessValue: brightness, redValue: red, greenValue: green, blueValue: blue,
		appliedBrightness: appliedBrightness, appliedRed: appliedRed,
		appliedGreen: appliedGreen, appliedBlue: appliedBlue,
		up: up, down: down, upPressedAt: upPressedAt, upHeld: upHeld,
	}
	r.initialize(font, page)
	if show {
		print("PAPERMONO READER SETTINGS OPEN\n")
	} else {
		print("PAPERMONO READER SETTINGS CLOSED\n")
	}
	return true
}

func (reader *reader) applySettings() error {
	reader.syncSettingsValues()
	brightness := reader.brightnessValue
	red, green, blue := reader.redValue, reader.greenValue, reader.blueValue
	if brightness == reader.appliedBrightness && red == reader.appliedRed &&
		green == reader.appliedGreen && blue == reader.appliedBlue {
		return nil
	}
	if brightness != reader.appliedBrightness {
		if err := board.Power.SetFrontLight(uint8(brightness)); err != nil {
			return err
		}
		reader.appliedBrightness = brightness
	}
	if red == reader.appliedRed && green == reader.appliedGreen && blue == reader.appliedBlue {
		return nil
	}
	if err := board.Power.SetRGB(uint8(red), uint8(green), uint8(blue)); err != nil {
		return err
	}
	reader.appliedRed, reader.appliedGreen, reader.appliedBlue = red, green, blue
	return nil
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
	wasUp := reader.up.stable
	reader.up.update(up, now)
	if !wasUp && reader.up.stable {
		reader.upPressedAt = now
		reader.upHeld = false
	}
	if reader.up.stable && !reader.upHeld && now-reader.upPressedAt >= settingsHoldMS {
		reader.upHeld = true
		reader.showSettings(!reader.settings)
	}
	if wasUp && !reader.up.stable && !reader.upHeld && !reader.settings {
		reader.queuePage(-1)
	}
	if reader.down.update(down, now) && !reader.settings {
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

func (reader *reader) samplePointer(x, y int, pressed bool) {
	changed := pressed != reader.sampledPressed
	if pressed && (x != reader.sampledX || y != reader.sampledY) {
		changed = true
	}
	if !changed {
		return
	}
	if pressed {
		reader.sampledX, reader.sampledY = x, y
	}
	event := pointerEvent{x: reader.sampledX, y: reader.sampledY, pressed: pressed}
	if reader.eventCount < len(reader.events) {
		reader.events[reader.eventCount] = event
		reader.eventCount++
	} else {
		reader.events[len(reader.events)-1] = event
	}
	reader.sampledPressed = pressed
}

func (reader *reader) captureTouch() error {
	point, pressed, err := board.Touch.Read()
	if err != nil {
		return err
	}
	reader.samplePointer(point.X, point.Y, pressed)
	return nil
}

func (reader *reader) dispatchCaptured() {
	for index := 0; index < reader.eventCount; index++ {
		event := reader.events[index]
		if event.pressed && !reader.dispatchedDown {
			reader.form.Dispatch(graphics.Event{Type: graphics.EventPointerDown,
				X: graphics.Scalar(event.x), Y: graphics.Scalar(event.y), Button: 1})
			reader.dispatchedDown = true
		} else if event.pressed && (event.x != reader.dispatchedX || event.y != reader.dispatchedY) {
			reader.form.Dispatch(graphics.Event{Type: graphics.EventPointerMove,
				X: graphics.Scalar(event.x), Y: graphics.Scalar(event.y), Button: 1})
		} else if !event.pressed && reader.dispatchedDown {
			reader.form.Dispatch(graphics.Event{Type: graphics.EventPointerUp,
				X: graphics.Scalar(reader.dispatchedX), Y: graphics.Scalar(reader.dispatchedY), Button: 1})
			reader.dispatchedDown = false
		}
		if event.pressed {
			reader.dispatchedX, reader.dispatchedY = event.x, event.y
		}
	}
	reader.eventCount = 0
}

func (reader *reader) touchActive() bool {
	return reader.sampledPressed || reader.dispatchedDown
}

func (reader *reader) PollDuringRefresh() error {
	reader.sampleButtons()
	return reader.captureTouch()
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
	if err := board.Power.SetRGB(0, 0, 0); err != nil {
		fail("RENVO PAPERMONO-LITE READER RGB FAIL\n")
	}
	board.Clock.DelayMilliseconds(300)
	if _, err := board.Touch.Initialize(); err != nil {
		fail("RENVO PAPERMONO-LITE READER TOUCH FAIL\n")
	}

	frame.Fill(true)
	surface := graphics.NewSurfaceBufferFormatPreserve(
		ssd1677.Width, ssd1677.Height, graphics.PixelMono1, frame[:])
	if surface == nil {
		fail("RENVO PAPERMONO-LITE READER SURFACE FAIL\n")
	}
	surface.SetAffine(0, -1, 1, 0, 0, ssd1677.Height)

	app := reader{brightnessValue: frontLight, appliedBrightness: frontLight}
	uiLowMark, uiHighMark := arena.Mark(), arena.PersistMark()
	app.initialize(font, 0)
	frameLowMark, frameHighMark := arena.Mark(), arena.PersistMark()
	app.sampleButtons()
	if !app.form.Paint(surface) {
		fail("RENVO PAPERMONO-LITE READER PAINT FAIL\n")
	}
	if err := board.Display.FullMonochrome(frame[:]); err != nil {
		fail("RENVO PAPERMONO-LITE READER DISPLAY FAIL\n")
	}
	surface.ResetDirty()
	arena.Reset(frameLowMark)
	arena.PersistReset(frameHighMark)
	print("RENVO PAPERMONO-LITE READER READY\n")

	for {
		app.sampleButtons()
		if app.rebuildRequestedView(uiLowMark, uiHighMark) {
			frameLowMark, frameHighMark = arena.Mark(), arena.PersistMark()
		}
		if err := app.captureTouch(); err != nil {
			fail("RENVO PAPERMONO-LITE READER TOUCH READ FAIL\n")
		}
		app.dispatchCaptured()
		if err := app.applySettings(); err != nil {
			fail("RENVO PAPERMONO-LITE READER SETTINGS FAIL\n")
		}
		app.applyPendingPage()
		if !app.touchActive() && app.form.InvalidRectCount() != 0 && app.form.Paint(surface) {
			if err := board.Display.FastMonochrome(frame[:], &app); err != nil {
				fail("RENVO PAPERMONO-LITE READER PRESENT FAIL\n")
			}
			surface.ResetDirty()
			app.dispatchCaptured()
			if err := app.applySettings(); err != nil {
				fail("RENVO PAPERMONO-LITE READER SETTINGS FAIL\n")
			}
		}
		arena.Reset(frameLowMark)
		arena.PersistReset(frameHighMark)
		board.Clock.DelayMilliseconds(1)
	}
}
