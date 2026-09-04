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
	screenWidth           = 480
	screenHeight          = 800
	maximumQueuedPointers = 64
	frontLightIdle        = 192
)

const (
	pageInputs = iota
	pageLists
	pageMotion
	pageMore
)

var frame ssd1677.Monochrome

type pointerEvent struct {
	x, y    int
	pressed bool
}

type showcase struct {
	form                  forms.Form
	bodyFont              *graphics.Font
	titleFont             *graphics.Font
	tabs                  *forms.TabControl
	status                *forms.StatusBar
	inputPage             []*forms.Control
	listPage              []*forms.Control
	motionPage            []*forms.Control
	morePage              []*forms.Control
	text                  *forms.TextBox
	area                  *forms.TextArea
	check                 *forms.CheckBox
	radioA                *forms.RadioButton
	combo                 *forms.ComboBox
	list                  *forms.ListBox
	progress              *forms.ProgressBar
	slider                *forms.Slider
	number                *forms.NumericUpDown
	split                 *forms.SplitContainer
	syncing               bool
	requestFull           bool
	currentPage           int
	requestedPage         int
	uiLowMark, uiHighMark int

	events                         [maximumQueuedPointers]pointerEvent
	eventCount                     int
	sampledX, sampledY             int
	sampledPressed, dispatchedDown bool
	dispatchedX, dispatchedY       int
}

func monochromeTheme() forms.Theme {
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

func (demo *showcase) add(page int, control *forms.Control) {
	demo.form.Add(control)
	if page == pageInputs {
		demo.inputPage = append(demo.inputPage, control)
	} else if page == pageLists {
		demo.listPage = append(demo.listPage, control)
	} else if page == pageMotion {
		demo.motionPage = append(demo.motionPage, control)
	} else {
		demo.morePage = append(demo.morePage, control)
	}
}

func setPageVisible(controls []*forms.Control, visible bool) {
	for _, control := range controls {
		control.SetVisible(visible)
	}
}

func (demo *showcase) setStatus(text string) {
	if demo.status.Text() == text {
		return
	}
	demo.status.SetText(text)
}

func (demo *showcase) showSelectedPage() {
	selected := demo.tabs.SelectedIndex()
	if selected != demo.currentPage {
		demo.requestedPage = selected
		return
	}
	setPageVisible(demo.inputPage, selected == pageInputs)
	setPageVisible(demo.listPage, selected == pageLists)
	setPageVisible(demo.motionPage, selected == pageMotion)
	setPageVisible(demo.morePage, selected == pageMore)
	if selected == pageInputs {
		demo.setStatus("Inputs: tap every control")
	} else if selected == pageLists {
		demo.setStatus("Lists: select and scroll")
	} else if selected == pageMotion {
		demo.setStatus("Motion: drag slider and divider")
	} else {
		demo.setStatus("More: toolbar, image and panel")
	}
}

func (demo *showcase) selectPage(page int) {
	demo.tabs.SetSelectedIndex(page)
	demo.showSelectedPage()
}

func (demo *showcase) rebuildRequestedPage() bool {
	page := demo.requestedPage
	if page < pageInputs || page > pageMore || demo.sampledPressed || demo.dispatchedDown {
		return false
	}
	bodyFont, titleFont := demo.bodyFont, demo.titleFont
	lowMark, highMark := demo.uiLowMark, demo.uiHighMark
	sampledX, sampledY := demo.sampledX, demo.sampledY
	arena.Reset(lowMark)
	arena.PersistReset(highMark)
	*demo = showcase{
		bodyFont: bodyFont, titleFont: titleFont,
		currentPage: page, requestedPage: -1,
		uiLowMark: lowMark, uiHighMark: highMark,
		sampledX: sampledX, sampledY: sampledY,
	}
	demo.initialize(bodyFont, titleFont, page)
	return true
}

func (demo *showcase) requestCleanRefresh() {
	demo.requestFull = true
	demo.form.Invalidate(graphics.R(0, 0, screenWidth, screenHeight))
	demo.setStatus("Full refresh requested")
}

func (demo *showcase) showAbout()    { demo.setStatus("Renvo Forms / SSD1677 / FT6336G") }
func (demo *showcase) selectInputs() { demo.selectPage(pageInputs) }
func (demo *showcase) selectLists()  { demo.selectPage(pageLists) }
func (demo *showcase) selectMotion() { demo.selectPage(pageMotion) }
func (demo *showcase) selectMore()   { demo.selectPage(pageMore) }

func (demo *showcase) editText() {
	demo.text.SetText("PaperMono text field")
	demo.setStatus("TextBox activated")
}

func (demo *showcase) editArea() {
	demo.area.SetText("TrueType cache\n480 x 800 at one bit")
	demo.setStatus("TextArea activated")
}

func (demo *showcase) checkChanged() {
	if demo.check.Checked() {
		demo.setStatus("CheckBox checked")
	} else {
		demo.setStatus("CheckBox cleared")
	}
}

func (demo *showcase) chooseRadioA() {
	demo.radioA.SetChecked(true)
	demo.setStatus("Radio: comfortable")
}

func (demo *showcase) advanceProgress() {
	value := demo.progress.Value() + 10
	if value > 100 {
		value = 0
	}
	demo.progress.SetValue(value)
	demo.setStatus("Button advanced progress")
}

func (demo *showcase) comboChanged() { demo.setStatus("Combo selection changed") }
func (demo *showcase) listChanged()  { demo.setStatus("List selection changed") }

func (demo *showcase) sliderChanged() {
	if demo.syncing {
		return
	}
	demo.syncing = true
	if demo.progress != nil {
		demo.progress.SetValue(demo.slider.Value())
	}
	demo.number.SetValue(demo.slider.Value())
	demo.syncing = false
	demo.setStatus("Slider adjusted")
}

func (demo *showcase) numberChanged() {
	if demo.syncing {
		return
	}
	demo.syncing = true
	demo.slider.SetValue(demo.number.Value())
	if demo.progress != nil {
		demo.progress.SetValue(demo.number.Value())
	}
	demo.syncing = false
	demo.setStatus("Stepper adjusted")
}

func (demo *showcase) splitChanged() { demo.setStatus("Split divider moved") }

func (demo *showcase) buildMenu() {
	bar := forms.NewMenuBar()
	bar.SetBounds(graphics.R(0, 0, screenWidth, 32))
	bar.SetFont(demo.bodyFont)

	display := forms.NewMenu("Display")
	clean := forms.NewMenuItem("Clean refresh")
	clean.Activate = demo.requestCleanRefresh
	display.Add(clean)
	display.Add(forms.NewMenuSeparator())
	about := forms.NewMenuItem("About PaperMono")
	about.Activate = demo.showAbout
	display.Add(about)
	bar.Add(display)
	demo.form.Add(&bar.Control)
	demo.form.SetMenuBar(bar)
}

func (demo *showcase) buildInputsPage() {
	group := forms.NewGroupBox()
	group.SetBounds(graphics.R(10, 150, 460, 596))
	group.SetFont(demo.bodyFont)
	group.SetText("Input controls")
	demo.add(pageInputs, &group.Control)

	demo.text = forms.NewTextBox()
	demo.text.SetBounds(graphics.R(26, 178, 428, 54))
	demo.text.SetFont(demo.bodyFont)
	demo.text.SetText("Tap to edit")
	demo.text.Click = demo.editText
	demo.add(pageInputs, &demo.text.Control)

	demo.area = forms.NewTextArea()
	demo.area.SetBounds(graphics.R(26, 246, 428, 92))
	demo.area.SetFont(demo.bodyFont)
	demo.area.SetText("Retained Forms\npacked monochrome surface")
	demo.area.Click = demo.editArea
	demo.add(pageInputs, &demo.area.Control)

	demo.check = forms.NewCheckBox()
	demo.check.SetBounds(graphics.R(26, 354, 428, 48))
	demo.check.SetFont(demo.bodyFont)
	demo.check.SetText("Enable notifications")
	demo.check.SetChecked(true)
	demo.check.Click = demo.checkChanged
	demo.add(pageInputs, &demo.check.Control)

	demo.radioA = forms.NewRadioButton()
	demo.radioA.SetBounds(graphics.R(26, 414, 428, 48))
	demo.radioA.SetFont(demo.bodyFont)
	demo.radioA.SetText("Comfortable")
	demo.radioA.SetChecked(true)
	demo.radioA.Click = demo.chooseRadioA
	demo.add(pageInputs, &demo.radioA.Control)

	button := forms.NewButton()
	button.SetBounds(graphics.R(26, 478, 428, 54))
	button.SetFont(demo.bodyFont)
	button.SetText("Advance progress")
	button.Click = demo.advanceProgress
	demo.add(pageInputs, &button.Control)

	demo.progress = forms.NewProgressBar()
	demo.progress.SetBounds(graphics.R(26, 554, 428, 28))
	demo.progress.SetValue(35)
	demo.add(pageInputs, &demo.progress.Control)
}

func (demo *showcase) buildListsPage() {
	demo.combo = forms.NewComboBox()
	demo.combo.SetBounds(graphics.R(18, 164, 444, 52))
	demo.combo.SetFont(demo.bodyFont)
	demo.combo.SetMaxDropDownItems(5)
	for _, item := range []string{"Brisbane", "Melbourne", "Tokyo"} {
		demo.combo.AddItem(item)
	}
	demo.combo.SetSelectedIndex(0)
	demo.combo.Changed = demo.comboChanged
	demo.add(pageLists, &demo.combo.Control)

	demo.list = forms.NewListBox()
	demo.list.SetBounds(graphics.R(18, 236, 210, 214))
	demo.list.SetFont(demo.bodyFont)
	for _, item := range []string{"Button", "TextBox", "CheckBox", "ComboBox"} {
		demo.list.AddItem(item)
	}
	demo.list.SetSelectedIndex(0)
	demo.list.Changed = demo.listChanged
	demo.add(pageLists, &demo.list.Control)

	tree := forms.NewTreeView()
	tree.SetBounds(graphics.R(246, 236, 216, 214))
	tree.SetFont(demo.bodyFont)
	tree.AddNode("Forms", 0)
	tree.AddNode("Inputs", 1)
	tree.AddNode("Lists", 1)
	tree.AddNode("Graphics", 0)
	tree.AddNode("TrueType", 1)
	demo.add(pageLists, &tree.Control)

	listView := forms.NewListView()
	listView.SetBounds(graphics.R(18, 470, 444, 242))
	listView.SetFont(demo.bodyFont)
	listView.AddColumn("Control")
	listView.AddColumn("Gesture")
	listView.SetColumnWidth(0, 260)
	listView.AddRow([]string{"Button", "Tap"})
	listView.AddRow([]string{"Slider", "Drag"})
	demo.add(pageLists, &listView.Control)
}

func (demo *showcase) buildMotionPage() {
	demo.slider = forms.NewSlider()
	demo.slider.SetBounds(graphics.R(18, 174, 444, 78))
	demo.slider.SetValue(35)
	demo.slider.Changed = demo.sliderChanged
	demo.add(pageMotion, &demo.slider.Control)

	demo.number = forms.NewNumericUpDown()
	demo.number.SetBounds(graphics.R(18, 282, 444, 58))
	demo.number.SetFont(demo.bodyFont)
	demo.number.SetValue(35)
	demo.number.SetIncrement(5)
	demo.number.Changed = demo.numberChanged
	demo.add(pageMotion, &demo.number.Control)

	demo.split = forms.NewSplitContainer()
	demo.split.SetBounds(graphics.R(18, 372, 444, 260))
	demo.split.SetPanelMinimumSizes(80, 80)
	demo.split.SetSplitterDistance(220)
	demo.split.Changed = demo.splitChanged
	demo.add(pageMotion, &demo.split.Control)
}

func (demo *showcase) buildMorePage() {
	toolbar := forms.NewToolBar()
	toolbar.SetBounds(graphics.R(10, 154, 460, 54))
	toolbar.SetFont(demo.bodyFont)
	toolbar.AddButton("Inputs", demo.selectInputs)
	toolbar.AddButton("Clean", demo.requestCleanRefresh)
	demo.add(pageMore, &toolbar.Control)

	picture := forms.NewPictureBox()
	picture.SetBounds(graphics.R(18, 232, 210, 210))
	picture.SetAccessibilityName("Monochrome image placeholder")
	demo.add(pageMore, &picture.Control)

	panel := forms.NewPanel()
	panel.SetBounds(graphics.R(246, 232, 216, 210))
	demo.add(pageMore, &panel.Control)
}

func (demo *showcase) initialize(bodyFont, titleFont *graphics.Font, page int) {
	demo.bodyFont, demo.titleFont = bodyFont, titleFont
	demo.currentPage = page
	demo.requestedPage = -1
	demo.form.Initialize(screenWidth, screenHeight)
	demo.form.ApplyTheme(monochromeTheme())
	demo.buildMenu()

	title := forms.NewLabel()
	title.SetBounds(graphics.R(16, 38, 448, 32))
	title.SetFont(titleFont)
	title.SetText("RENVO PAPERMONO FORMS")
	demo.form.Add(&title.Control)
	demo.tabs = forms.NewTabControl()
	demo.tabs.SetBounds(graphics.R(0, 88, screenWidth, 56))
	demo.tabs.SetFont(bodyFont)
	demo.tabs.AddTab("Inputs")
	demo.tabs.AddTab("Lists")
	demo.tabs.AddTab("Motion")
	demo.tabs.AddTab("More")
	demo.tabs.SetSelectedIndex(page)
	demo.tabs.Changed = demo.showSelectedPage
	demo.form.Add(&demo.tabs.Control)

	demo.status = forms.NewStatusBar()
	demo.status.SetBounds(graphics.R(0, 758, screenWidth, 42))
	demo.status.SetFont(bodyFont)
	demo.form.Add(&demo.status.Control)

	if page == pageInputs {
		demo.buildInputsPage()
	} else if page == pageLists {
		demo.buildListsPage()
	} else if page == pageMotion {
		demo.buildMotionPage()
	} else {
		demo.buildMorePage()
	}
	demo.showSelectedPage()
}

func (demo *showcase) samplePointer(x, y int, pressed bool) {
	changed := pressed != demo.sampledPressed
	if pressed && (x != demo.sampledX || y != demo.sampledY) {
		changed = true
	}
	if !changed {
		return
	}
	if pressed {
		demo.sampledX, demo.sampledY = x, y
	}
	event := pointerEvent{x: demo.sampledX, y: demo.sampledY, pressed: pressed}
	if demo.eventCount < len(demo.events) {
		demo.events[demo.eventCount] = event
		demo.eventCount++
	} else {
		demo.events[len(demo.events)-1] = event
	}
	demo.sampledPressed = pressed
}

func (demo *showcase) capture() error {
	point, pressed, err := board.Touch.Read()
	if err != nil {
		return err
	}
	demo.samplePointer(point.X, point.Y, pressed)
	return nil
}

func (demo *showcase) PollDuringRefresh() error { return demo.capture() }

func (demo *showcase) touchActive() bool {
	return demo.sampledPressed || demo.dispatchedDown
}

func (demo *showcase) dispatchCaptured() {
	for index := 0; index < demo.eventCount; index++ {
		event := demo.events[index]
		if event.pressed && !demo.dispatchedDown {
			demo.form.Dispatch(graphics.Event{Type: graphics.EventPointerDown,
				X: graphics.Scalar(event.x), Y: graphics.Scalar(event.y), Button: 1})
			demo.dispatchedDown = true
		} else if event.pressed && (event.x != demo.dispatchedX || event.y != demo.dispatchedY) {
			demo.form.Dispatch(graphics.Event{Type: graphics.EventPointerMove,
				X: graphics.Scalar(event.x), Y: graphics.Scalar(event.y), Button: 1})
		} else if !event.pressed && demo.dispatchedDown {
			demo.form.Dispatch(graphics.Event{Type: graphics.EventPointerUp,
				X: graphics.Scalar(demo.dispatchedX), Y: graphics.Scalar(demo.dispatchedY), Button: 1})
			demo.dispatchedDown = false
		}
		if event.pressed {
			demo.dispatchedX, demo.dispatchedY = event.x, event.y
		}
	}
	demo.eventCount = 0
}

func fail(message string) {
	print(message)
	_ = board.Display.Shutdown()
	for {
	}
}

func main() {
	bodyFont := fontcache.TitleSubset(fontcache.PaperMonoFormsGlyphs)
	titleFont := bodyFont
	if bodyFont == nil {
		print("RENVO PAPERMONO-LITE FORMS FONT FAIL\n")
		for {
		}
	}
	print("PAPERMONO FORMS FONT READY\n")
	if board.Initialize() != nil {
		fail("RENVO PAPERMONO-LITE FORMS BOARD FAIL\n")
	}
	if err := board.Display.Enable(); err != nil {
		print("RENVO PAPERMONO-LITE FORMS POWER FAIL: ", err.Error(), "\n")
		for {
		}
	}
	if err := board.Power.SetFrontLight(frontLightIdle); err != nil {
		fail("RENVO PAPERMONO-LITE FORMS FRONTLIGHT FAIL\n")
	}
	print("PAPERMONO FORMS FRONTLIGHT READY\n")
	// The display and FT6336G share the switched board power sequence. Give the
	// touch controller time to boot after reset before reading its identity.
	board.Clock.DelayMilliseconds(300)
	identity, err := board.Touch.Initialize()
	if err != nil {
		print("RENVO PAPERMONO-LITE FORMS TOUCH FAIL: ", err.Error(), "\n")
		_ = board.Display.Shutdown()
		for {
		}
	}
	print("PAPERMONO FORMS TOUCH READY\n")
	frame.Fill(true)
	surface := graphics.NewSurfaceBufferFormatPreserve(ssd1677.Width, ssd1677.Height, graphics.PixelMono1, frame[:])
	if surface == nil {
		fail("RENVO PAPERMONO-LITE FORMS SURFACE FAIL\n")
	}
	// Logical portrait edges map to native panel edges as (x,y)->(y,480-x).
	surface.SetAffine(0, -1, 1, 0, 0, ssd1677.Height)
	var demo showcase
	demo.uiLowMark, demo.uiHighMark = arena.Mark(), arena.PersistMark()
	demo.initialize(bodyFont, titleFont, pageInputs)
	print("PAPERMONO FORMS CONTROLS READY\n")
	if !demo.form.Paint(surface) {
		fail("RENVO PAPERMONO-LITE FORMS PAINT FAIL\n")
	}
	if err := board.Display.FullMonochrome(frame[:]); err != nil {
		fail("RENVO PAPERMONO-LITE FORMS DISPLAY FAIL\n")
	}
	surface.ResetDirty()
	print("RENVO PAPERMONO-LITE FORMS READY CIPHER=", identity.Cipher,
		" FIRMWARE=", identity.Firmware, " VENDOR=", identity.Vendor, "\n")

	for {
		if err := demo.capture(); err != nil {
			fail("RENVO PAPERMONO-LITE FORMS READ FAIL\n")
		}
		demo.dispatchCaptured()
		demo.rebuildRequestedPage()
		// A short e-paper tap otherwise spends one refresh showing a transient
		// pressed state and a second refresh showing the action. Keep collecting
		// and dispatching touch samples while the finger is down, then paint the
		// final state once on release. Drag controls still receive every move and
		// retain their final value.
		if demo.touchActive() {
			board.Clock.DelayMilliseconds(1)
			continue
		}
		if demo.form.Paint(surface) {
			if demo.requestFull {
				demo.requestFull = false
				if err := board.Display.FullMonochrome(frame[:]); err != nil {
					fail("RENVO PAPERMONO-LITE FORMS FULL FAIL\n")
				}
			} else if err := board.Display.FastMonochrome(frame[:], &demo); err != nil {
				fail("RENVO PAPERMONO-LITE FORMS PRESENT FAIL\n")
			}
			surface.ResetDirty()
			demo.dispatchCaptured()
			demo.rebuildRequestedPage()
		}
		board.Clock.DelayMilliseconds(1)
	}
}
