package main

import (
	"renvo.dev/device/board"
	"renvo.dev/examples/device/fontcache"
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
	"renvo.dev/std/strconv"
)

const (
	width  = 720
	height = 1280
)

type touchOverlay struct {
	forms.Control
	points             [10]board.TouchPoint
	count              int
	font               *graphics.Font
	target             *forms.TextBox
	keyboardVisible    bool
	keyboardShift      bool
	keyboardPressed    int
	keyboardPrevious   int
	keyboardFullPasses int
}

func newTouchOverlay(font *graphics.Font, target *forms.TextBox) *touchOverlay {
	overlay := &touchOverlay{font: font, target: target, keyboardPressed: -1, keyboardPrevious: -1}
	overlay.Control = *forms.NewControl()
	overlay.SetBounds(graphics.R(0, 0, width, height))
	overlay.SetEnabled(false)
	overlay.SetTabStop(false)
	overlay.Paint = overlay.paint
	return overlay
}

func touchRect(point board.TouchPoint) graphics.Rect {
	x, y := board.PortraitPoint(point)
	return graphics.R(graphics.Scalar(x-18), graphics.Scalar(y-18), 36, 36)
}

func (overlay *touchOverlay) update(points []board.TouchPoint, count int) {
	unchanged := overlay.count == count
	if unchanged {
		for index := 0; index < count; index++ {
			old := overlay.points[index]
			current := points[index]
			if old.ID != current.ID || old.X != current.X || old.Y != current.Y {
				unchanged = false
				break
			}
		}
	}
	if unchanged {
		return
	}
	form := overlay.Form()
	if form != nil {
		for index := 0; index < overlay.count; index++ {
			form.Invalidate(touchRect(overlay.points[index]))
		}
		for index := 0; index < count; index++ {
			form.Invalidate(touchRect(points[index]))
		}
	}
	overlay.count = count
	for index := 0; index < count; index++ {
		overlay.points[index] = points[index]
	}
}

func (overlay *touchOverlay) paint(canvas graphics.Canvas) {
	if overlay.keyboardVisible {
		overlay.paintKeyboard(canvas)
	}
	colors := []graphics.Color{
		graphics.RGBA(40, 180, 255, 208), graphics.RGBA(255, 88, 120, 208),
		graphics.RGBA(80, 220, 150, 208), graphics.RGBA(255, 190, 55, 208),
	}
	for index := 0; index < overlay.count; index++ {
		x, y := board.PortraitPoint(overlay.points[index])
		canvas.FillEllipse(graphics.R(graphics.Scalar(x-14), graphics.Scalar(y-14), 28, 28), colors[index%len(colors)])
	}
}

type controlsDemo struct {
	form            forms.Form
	status          *forms.Label
	progress        *forms.ProgressBar
	slider          *forms.Slider
	number          *forms.NumericUpDown
	check           *forms.CheckBox
	radioA          *forms.RadioButton
	radioB          *forms.RadioButton
	text            *forms.TextBox
	overlay         *touchOverlay
	fps             *forms.Label
	dark            bool
	primary         int
	primaryX        int
	primaryY        int
	multiSeen       bool
	readError       int
	lastReport      int
	syncing         bool
	fpsTime         uint32
	fpsFrame        uint32
	keyboardCapture bool
}

func (demo *controlsDemo) setStatus(text string) {
	demo.status.SetText(text)
}

func (demo *controlsDemo) tabChanged() {
	demo.setStatus("Tab selection changed")
}

func (demo *controlsDemo) textFocused(x, y graphics.Scalar) {
	demo.overlay.showKeyboard()
	demo.setStatus("Text field focused")
}

func (demo *controlsDemo) comboChanged() {
	demo.setStatus("Combo selection changed")
}

func (demo *controlsDemo) listChanged() {
	demo.setStatus("List selection changed")
}

func (demo *controlsDemo) advance() {
	value := demo.progress.Value() + 10
	if value > 100 {
		value = 0
	}
	demo.progress.SetValue(value)
	demo.slider.SetValue(value)
	demo.number.SetValue(value)
	demo.setStatus("Button: progress advanced")
}

func (demo *controlsDemo) checkChanged() {
	if demo.check.Checked() {
		demo.setStatus("Checkbox enabled")
	} else {
		demo.setStatus("Checkbox disabled")
	}
}

func (demo *controlsDemo) chooseA() {
	demo.radioA.SetChecked(true)
	demo.radioB.SetChecked(false)
	demo.setStatus("Density: comfortable")
}

func (demo *controlsDemo) chooseB() {
	demo.radioA.SetChecked(false)
	demo.radioB.SetChecked(true)
	demo.setStatus("Density: compact")
}

func (demo *controlsDemo) sliderChanged() {
	if demo.syncing {
		return
	}
	demo.syncing = true
	demo.progress.SetValue(demo.slider.Value())
	demo.number.SetValue(demo.slider.Value())
	demo.syncing = false
	demo.setStatus("Slider: continuous drag")
}

func (demo *controlsDemo) numberChanged() {
	if demo.syncing {
		return
	}
	demo.syncing = true
	demo.progress.SetValue(demo.number.Value())
	demo.slider.SetValue(demo.number.Value())
	demo.syncing = false
	demo.setStatus("Stepper value changed")
}

func (demo *controlsDemo) toggleTheme() {
	demo.dark = !demo.dark
	if demo.dark {
		demo.form.ApplyTheme(forms.DarkTheme())
		demo.setStatus("Dark theme")
	} else {
		demo.form.ApplyTheme(forms.LightTheme())
		demo.setStatus("Light theme")
	}
}

func (demo *controlsDemo) updateFPS() {
	frame := board.FrameNumber()
	if frame == demo.fpsFrame {
		return
	}
	now := board.Milliseconds()
	if demo.fpsTime == 0 {
		demo.fpsTime = now
		demo.fpsFrame = frame
		return
	}
	elapsed := now - demo.fpsTime
	if elapsed < 1000 {
		return
	}
	tenths := int((frame - demo.fpsFrame) * 10000 / elapsed)
	demo.fps.SetText(strconv.Itoa(tenths/10) + "." + strconv.Itoa(tenths%10) + " FPS")
	demo.fpsTime = now
	demo.fpsFrame = frame
}

func addLabel(form *forms.Form, font *graphics.Font, text string, bounds graphics.Rect) *forms.Label {
	label := forms.NewLabel()
	label.SetBounds(bounds)
	label.SetFont(font)
	label.SetText(text)
	form.Add(&label.Control)
	return label
}

func (demo *controlsDemo) initialize(font, titleFont *graphics.Font) {
	demo.form.Initialize(width, height)
	demo.form.ApplyTheme(forms.LightTheme())
	demo.primary = -1
	demo.lastReport = -1

	addLabel(&demo.form, titleFont, "RENVO TAB5 CONTROLS", graphics.R(20, 12, 680, 42))
	demo.fps = addLabel(&demo.form, titleFont, "--.- FPS", graphics.R(550, 12, 150, 42))
	addLabel(&demo.form, font, "ST7121  |  720 x 1280  |  10-point touch", graphics.R(20, 56, 680, 28))

	tabs := forms.NewTabControl()
	tabs.SetBounds(graphics.R(20, 90, 680, 48))
	tabs.SetFont(font)
	tabs.AddTab("Inputs")
	tabs.AddTab("Lists")
	tabs.AddTab("Motion")
	tabs.AddTab("More")
	tabs.Changed = demo.tabChanged
	demo.form.Add(&tabs.Control)

	inputGroup := forms.NewGroupBox()
	inputGroup.SetBounds(graphics.R(20, 154, 330, 500))
	inputGroup.SetFont(font)
	inputGroup.SetText("Touch inputs")
	demo.form.Add(&inputGroup.Control)

	addLabel(&demo.form, font, "Text field", graphics.R(40, 188, 290, 24))
	text := forms.NewTextBox()
	text.SetBounds(graphics.R(40, 216, 290, 48))
	text.SetFont(font)
	text.SetText("Tap to focus")
	text.PointerDown = demo.textFocused
	demo.text = text
	demo.form.Add(&text.Control)

	demo.check = forms.NewCheckBox()
	demo.check.SetBounds(graphics.R(40, 278, 290, 48))
	demo.check.SetFont(font)
	demo.check.SetText("Enable notifications")
	demo.check.SetChecked(true)
	demo.check.Click = demo.checkChanged
	demo.form.Add(&demo.check.Control)

	demo.radioA = forms.NewRadioButton()
	demo.radioA.SetBounds(graphics.R(40, 338, 145, 44))
	demo.radioA.SetFont(font)
	demo.radioA.SetText("Comfortable")
	demo.radioA.SetChecked(true)
	demo.radioA.Click = demo.chooseA
	demo.form.Add(&demo.radioA.Control)
	demo.radioB = forms.NewRadioButton()
	demo.radioB.SetBounds(graphics.R(185, 338, 145, 44))
	demo.radioB.SetFont(font)
	demo.radioB.SetText("Compact")
	demo.radioB.Click = demo.chooseB
	demo.form.Add(&demo.radioB.Control)

	button := forms.NewButton()
	button.SetBounds(graphics.R(40, 400, 290, 54))
	button.SetFont(font)
	button.SetText("Advance progress")
	button.Click = demo.advance
	demo.form.Add(&button.Control)
	demo.progress = forms.NewProgressBar()
	demo.progress.SetBounds(graphics.R(40, 468, 290, 24))
	demo.progress.SetValue(35)
	demo.form.Add(&demo.progress.Control)
	disabled := forms.NewButton()
	disabled.SetBounds(graphics.R(40, 508, 290, 52))
	disabled.SetFont(font)
	disabled.SetText("Disabled action")
	disabled.SetEnabled(false)
	demo.form.Add(&disabled.Control)
	theme := forms.NewButton()
	theme.SetBounds(graphics.R(40, 578, 290, 52))
	theme.SetFont(font)
	theme.SetText("Toggle light / dark theme")
	theme.Click = demo.toggleTheme
	demo.form.Add(&theme.Control)

	listGroup := forms.NewGroupBox()
	listGroup.SetBounds(graphics.R(370, 154, 330, 500))
	listGroup.SetFont(font)
	listGroup.SetText("Selection controls")
	demo.form.Add(&listGroup.Control)
	combo := forms.NewComboBox()
	combo.SetBounds(graphics.R(390, 188, 290, 48))
	combo.SetFont(font)
	combo.SetMaxDropDownItems(5)
	for _, item := range []string{"Brisbane", "Melbourne", "Sydney", "Auckland", "Tokyo", "Singapore"} {
		combo.AddItem(item)
	}
	combo.SetSelectedIndex(0)
	combo.Changed = demo.comboChanged
	demo.form.Add(&combo.Control)
	list := forms.NewListBox()
	list.SetBounds(graphics.R(390, 252, 290, 180))
	list.SetFont(font)
	for _, item := range []string{"Buttons", "Text fields", "Check boxes", "Radio buttons", "Combo boxes", "List boxes", "Tree views", "Sliders"} {
		list.AddItem(item)
	}
	list.SetSelectedIndex(0)
	list.Changed = demo.listChanged
	demo.form.Add(&list.Control)
	tree := forms.NewTreeView()
	tree.SetBounds(graphics.R(390, 450, 290, 184))
	tree.SetFont(font)
	tree.AddNode("Controls", 0)
	tree.AddNode("Inputs", 1)
	tree.AddNode("Selection", 1)
	tree.AddNode("Motion", 1)
	tree.AddNode("Graphics", 0)
	tree.AddNode("RGB565 scanout", 1)
	demo.form.Add(&tree.Control)

	motionGroup := forms.NewGroupBox()
	motionGroup.SetBounds(graphics.R(20, 674, 680, 514))
	motionGroup.SetFont(font)
	motionGroup.SetText("Motion and data")
	demo.form.Add(&motionGroup.Control)
	addLabel(&demo.form, font, "Drag slider", graphics.R(40, 708, 640, 24))
	demo.slider = forms.NewSlider()
	demo.slider.SetBounds(graphics.R(40, 738, 640, 64))
	demo.slider.SetValue(35)
	demo.slider.Changed = demo.sliderChanged
	demo.form.Add(&demo.slider.Control)
	addLabel(&demo.form, font, "Numeric stepper", graphics.R(40, 824, 300, 28))
	demo.number = forms.NewNumericUpDown()
	demo.number.SetBounds(graphics.R(390, 812, 290, 52))
	demo.number.SetFont(font)
	demo.number.SetValue(35)
	demo.number.SetIncrement(5)
	demo.number.Changed = demo.numberChanged
	demo.form.Add(&demo.number.Control)
	listView := forms.NewListView()
	listView.SetBounds(graphics.R(40, 886, 640, 190))
	listView.SetFont(font)
	listView.AddColumn("Control")
	listView.AddColumn("Gesture")
	listView.SetColumnWidth(0, 360)
	listView.AddRow([]string{"Button", "Tap"})
	listView.AddRow([]string{"ListBox", "Swipe"})
	listView.AddRow([]string{"Slider", "Drag"})
	listView.AddRow([]string{"Display", "Multi-touch"})
	demo.form.Add(&listView.Control)
	addLabel(&demo.form, font, "Colored markers show every active contact.", graphics.R(40, 1095, 640, 54))

	demo.status = addLabel(&demo.form, font, "Ready - touch a control", graphics.R(20, 1208, 680, 42))
	demo.overlay = newTouchOverlay(font, text)
	demo.form.Add(&demo.overlay.Control)
}

func (demo *controlsDemo) pollTouch() bool {
	var points [10]board.TouchPoint
	count, ok := board.ReadTouches(points[:])
	if !ok {
		failure := board.TouchReadFailure()
		if demo.readError != failure {
			demo.setStatus("Touch read error")
			if failure == 1 {
				print("TAB5 TOUCH STATUS READ FAIL\n")
			} else if failure == 2 {
				print("TAB5 TOUCH REPORT READ FAIL\n")
			}
			demo.readError = failure
		}
		return false
	}
	demo.readError = 0
	report := board.TouchLastReportStats().Reports
	if report == demo.lastReport {
		return true
	}
	demo.lastReport = report
	demo.overlay.update(points[:], count)
	primaryIndex := -1
	for index := 0; index < count; index++ {
		if points[index].ID == demo.primary {
			primaryIndex = index
		}
	}
	if demo.primary >= 0 && primaryIndex < 0 {
		if demo.keyboardCapture {
			demo.overlay.keyboardUp(graphics.Scalar(demo.primaryX), graphics.Scalar(demo.primaryY))
			demo.keyboardCapture = false
		} else {
			demo.form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: graphics.Scalar(demo.primaryX), Y: graphics.Scalar(demo.primaryY), Button: 1})
		}
		demo.primary = -1
	}
	if demo.primary < 0 && count > 0 {
		demo.primary = points[0].ID
		primaryIndex = 0
		x, y := board.PortraitPoint(points[0])
		demo.primaryX, demo.primaryY = x, y
		if demo.overlay.keyboardVisible && y >= keyboardTop && y < keyboardBottom {
			demo.keyboardCapture = true
			demo.overlay.keyboardDown(graphics.Scalar(x), graphics.Scalar(y))
		} else {
			demo.form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: graphics.Scalar(x), Y: graphics.Scalar(y), Button: 1})
		}
	} else if primaryIndex >= 0 {
		x, y := board.PortraitPoint(points[primaryIndex])
		demo.primaryX, demo.primaryY = x, y
		if demo.keyboardCapture {
			demo.overlay.keyboardMove(graphics.Scalar(x), graphics.Scalar(y))
		} else {
			demo.form.Dispatch(graphics.Event{Type: graphics.EventPointerMove, X: graphics.Scalar(x), Y: graphics.Scalar(y), Button: 1})
		}
	}
	if count > 1 {
		demo.setStatus("Multi-touch active: all contacts tracked")
		if !demo.multiSeen {
			print("TAB5 MULTITOUCH PASS\n")
			demo.multiSeen = true
		}
	}
	if demo.overlay.keyboardVisible && !demo.text.Focused() {
		demo.overlay.hideKeyboard()
	}
	return true
}

func main() {
	print("TAB5 FORMS MAIN\n")
	print("TAB5 FORMS ARENA PASS\n")
	if !board.InitFramebuffer() {
		print("TAB5 FORMS DISPLAY INIT FAIL\n")
		for {
		}
	}
	print("TAB5 FORMS DISPLAY INIT PASS\n")
	if !board.InitTouch() {
		print("TAB5 FORMS TOUCH INIT FAIL\n")
		for {
		}
	}
	print("TAB5 FORMS TOUCH INIT PASS\n")
	surface := board.NewPortraitSurface()
	if surface == nil {
		print("TAB5 FORMS SURFACE FAIL\n")
		for {
		}
	}
	print("TAB5 FORMS SURFACE PASS\n")
	// Own the cached font graphs in main's application-lifetime frame. Controls
	// retain these pointers without forcing the arena to persist-copy hundreds
	// of glyph masks when their initialization helper returns.
	font := fontcache.Body()
	titleFont := fontcache.Title()
	if font == nil || titleFont == nil {
		print("TAB5 FORMS FONT FAIL\n")
		for {
		}
	}
	var demo controlsDemo
	demo.initialize(font, titleFont)
	print("TAB5 FORMS CONTROLS PASS\n")
	if !demo.form.Paint(surface) {
		print("TAB5 FORMS PAINT FAIL\n")
		for {
		}
	}
	demo.overlay.afterPaint()
	print("TAB5 FORMS PAINT PASS\n")
	if !board.PresentPortrait(surface) {
		print("TAB5 FORMS PRESENT FAIL\n")
		for {
		}
	}
	surface.ResetDirty()
	print("TAB5 PORTRAIT FORMS PASS\n")
	stats := board.FramebufferStats()
	if stats.DMA2DCopies > 0 {
		print("TAB5 DMA2D PASS\n")
	} else {
		print("TAB5 DMA2D FAIL\n")
	}
	if stats.ScanoutUnderruns > 0 {
		print("TAB5 SCANOUT UNDERRUN\n")
	}
	// The first presentation made both buffers identical. Thereafter each back
	// buffer is one generation old, so repainting current + previous damage keeps
	// it correct without a synchronous post-flip framebuffer copy.
	var previousDamage [64]graphics.Rect
	previousCount := 0
	for {
		demo.pollTouch()
		board.Refresh()
		demo.updateFPS()
		currentCount := demo.form.InvalidRectCount()
		if currentCount == 0 {
			continue
		}
		var currentDamage [64]graphics.Rect
		for index := 0; index < currentCount; index++ {
			currentDamage[index], _ = demo.form.InvalidRectAt(index)
		}
		for index := 0; index < previousCount; index++ {
			demo.form.Invalidate(previousDamage[index])
		}
		if demo.form.Paint(surface) {
			demo.overlay.afterPaint()
			if !board.PresentPortraitRetained(surface) {
				print("TAB5 FORMS PRESENT FAIL\n")
				for {
				}
			}
			surface.ResetDirty()
		}
		previousCount = currentCount
		for index := 0; index < currentCount; index++ {
			previousDamage[index] = currentDamage[index]
		}
	}
}
