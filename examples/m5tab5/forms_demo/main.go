package main

import (
	"renvo.dev/examples/m5tab5/board"
	"renvo.dev/forms"
	"renvo.dev/internal/arena"
	"renvo.dev/std/graphics"
)

const (
	width  = 720
	height = 1280
)

type touchOverlay struct {
	forms.Control
	points [10]board.TouchPoint
	count  int
}

func newTouchOverlay() *touchOverlay {
	overlay := &touchOverlay{}
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
	form      forms.Form
	status    *forms.Label
	progress  *forms.ProgressBar
	slider    *forms.Slider
	number    *forms.NumericUpDown
	check     *forms.CheckBox
	radioA    *forms.RadioButton
	radioB    *forms.RadioButton
	overlay   *touchOverlay
	dark      bool
	primary   int
	primaryX  int
	primaryY  int
	multiSeen bool
	readError int
}

func (demo *controlsDemo) setStatus(text string) {
	demo.status.SetText(text)
}

func (demo *controlsDemo) tabChanged() {
	demo.setStatus("Tab selection changed")
}

func (demo *controlsDemo) textFocused() {
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
	demo.progress.SetValue(demo.slider.Value())
	demo.number.SetValue(demo.slider.Value())
	demo.setStatus("Slider: continuous drag")
}

func (demo *controlsDemo) numberChanged() {
	demo.progress.SetValue(demo.number.Value())
	demo.slider.SetValue(demo.number.Value())
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

func addLabel(form *forms.Form, font *graphics.Font, text string, bounds graphics.Rect) *forms.Label {
	label := forms.NewLabel()
	label.SetBounds(bounds)
	label.SetFont(font)
	label.SetText(text)
	form.Add(&label.Control)
	return label
}

func (demo *controlsDemo) initialize() {
	demo.form.Initialize(width, height)
	demo.form.ApplyTheme(forms.LightTheme())
	demo.primary = -1
	font := graphics.NewBuiltinFont(2)
	titleFont := graphics.NewBuiltinFont(3)

	addLabel(&demo.form, titleFont, "RENVO TAB5 CONTROLS", graphics.R(20, 12, 680, 42))
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
	text.Click = demo.textFocused
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
	demo.overlay = newTouchOverlay()
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
	demo.overlay.update(points[:], count)
	primaryIndex := -1
	for index := 0; index < count; index++ {
		if points[index].ID == demo.primary {
			primaryIndex = index
		}
	}
	if demo.primary >= 0 && primaryIndex < 0 {
		demo.form.Dispatch(graphics.Event{Type: graphics.EventPointerUp, X: graphics.Scalar(demo.primaryX), Y: graphics.Scalar(demo.primaryY), Button: 1})
		demo.primary = -1
	}
	if demo.primary < 0 && count > 0 {
		demo.primary = points[0].ID
		primaryIndex = 0
		x, y := board.PortraitPoint(points[0])
		demo.primaryX, demo.primaryY = x, y
		demo.form.Dispatch(graphics.Event{Type: graphics.EventPointerDown, X: graphics.Scalar(x), Y: graphics.Scalar(y), Button: 1})
	} else if primaryIndex >= 0 {
		x, y := board.PortraitPoint(points[primaryIndex])
		demo.primaryX, demo.primaryY = x, y
		demo.form.Dispatch(graphics.Event{Type: graphics.EventPointerMove, X: graphics.Scalar(x), Y: graphics.Scalar(y), Button: 1})
	}
	if count > 1 {
		demo.setStatus("Multi-touch active: all contacts tracked")
		if !demo.multiSeen {
			print("TAB5 MULTITOUCH PASS\n")
			demo.multiSeen = true
		}
	}
	return true
}

func main() {
	print("TAB5 FORMS MAIN\n")
	// Keep the compiler's bidirectional object arena away from the two PSRAM
	// framebuffers. The low end serves transient allocations while persistent
	// controls and strings grow down from the high end.
	arena.Reset(0x48800000)
	arena.PersistReset(0x49f00000)
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
	var demo controlsDemo
	demo.initialize()
	print("TAB5 FORMS CONTROLS PASS\n")
	if !demo.form.Paint(surface) || !board.PresentPortrait(surface) {
		print("TAB5 FORMS PRESENT FAIL\n")
		for {
		}
	}
	surface.ResetDirty()
	print("TAB5 PORTRAIT FORMS PASS\n")
	for {
		demo.pollTouch()
		if demo.form.Paint(surface) {
			if board.PresentPortrait(surface) {
				surface.ResetDirty()
			}
		}
		board.Refresh()
	}
}
