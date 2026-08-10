package main

import (
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
	"renvo.dev/std/graphics/gofont"
)

const (
	pageInputs = iota
	pageLists
	pageMotion
	pageMore
)

const (
	keyNone = 256 + iota
	keyShift
	keyBackspace
	keySymbols
	keySpace
	keyEnter
	keyDone
)

const keyboardMaximumTextBytes = 1024

type controlsDemo struct {
	form          forms.Form
	bodyFont      *graphics.Font
	titleFont     *graphics.Font
	tabs          *forms.TabControl
	status        *forms.StatusBar
	inputProgress *forms.ProgressBar
	textBox       *forms.TextBox
	textArea      *forms.TextArea
	progress      *forms.ProgressBar
	slider        *forms.Slider
	number        *forms.NumericUpDown
	list          *forms.ListBox
	combo         *forms.ComboBox
	radioA        *forms.RadioButton
	radioB        *forms.RadioButton
	check         *forms.CheckBox
	split         *forms.SplitContainer
	keyboard      *touchKeyboard
	inputPage     []*forms.Control
	listPage      []*forms.Control
	motionPage    []*forms.Control
	morePage      []*forms.Control
}

// touchKeyboard keeps NativeActivity text entry entirely in the Go client.
// NativeActivity has no Java InputConnection, so requesting a platform IME
// alone cannot deliver committed text. Physical/ADB key events still use the
// graphics key path; this keyboard supplies the touch path.
type touchKeyboard struct {
	forms.Control
	font      *graphics.Font
	target    *forms.Control
	multiline bool
	shiftMode int
	symbols   bool
	pressed   int
	Done      func()
}

func newTouchKeyboard(font *graphics.Font) *touchKeyboard {
	k := &touchKeyboard{font: font}
	k.Control = *forms.NewControl()
	k.SetTabStop(false)
	k.SetVisible(false)
	k.Paint = k.paint
	k.PointerDown = k.pointerDown
	k.PointerUp = k.pointerUp
	k.PointerCancel = k.pointerCancel
	k.PointerMove = k.pointerMove
	k.PointerLeave = k.pointerLeave
	return k
}

func (k *touchKeyboard) show(target *forms.Control, multiline bool) {
	if k.target != target {
		if k.target != nil {
			k.target.EndTextEdit(true)
		}
		target.BeginTextEdit(keyboardMaximumTextBytes)
	}
	k.target = target
	k.multiline = multiline
	k.shiftMode = 1
	k.symbols = false
	k.pressed = keyNone
	k.SetVisible(true)
}

func (k *touchKeyboard) hide() {
	if k.target != nil {
		k.target.EndTextEdit(true)
	}
	k.SetVisible(false)
	k.target = nil
	k.pressed = keyNone
}

func (k *touchKeyboard) append(text string) {
	if k.target != nil {
		k.target.AppendText(text)
	}
}

func (k *touchKeyboard) backspace() {
	if k.target != nil {
		k.target.BackspaceText()
	}
}

func keyboardRowKey(row string, x, start, width int) int {
	if x < start || x >= start+width || len(row) == 0 {
		return keyNone
	}
	index := (x - start) * len(row) / width
	return int(row[index])
}

func (k *touchKeyboard) keyAt(x, y graphics.Scalar) int {
	ix, iy := int(x), int(y)
	row := (iy - 4) / 76
	if iy < 4 || row < 0 || row > 3 || iy-row*76 > 72 {
		return keyNone
	}
	row0, row1, row2 := "qwertyuiop", "asdfghjkl", "zxcvbnm"
	if k.symbols {
		row0, row1, row2 = "1234567890", "@#$%&-+()", "*\"':;!?"
	}
	if row == 0 {
		return keyboardRowKey(row0, ix, 4, 352)
	} else if row == 1 {
		return keyboardRowKey(row1, ix, 21, 318)
	} else if row == 2 {
		if ix >= 4 && ix < 52 {
			return keyShift
		}
		if ix >= 308 && ix < 356 {
			return keyBackspace
		}
		return keyboardRowKey(row2, ix, 56, 248)
	}
	if ix >= 4 && ix < 56 {
		return keySymbols
	}
	if ix >= 60 && ix < 98 {
		return int(',')
	}
	if ix >= 102 && ix < 242 {
		if !k.multiline || ix < 202 {
			return keySpace
		}
	}
	if k.multiline && ix >= 206 && ix < 240 {
		return int('.')
	}
	if !k.multiline && ix >= 246 && ix < 284 {
		return int('.')
	}
	if k.multiline && ix >= 244 && ix < 296 {
		return keyEnter
	}
	if k.multiline && ix >= 300 && ix < 356 {
		return keyDone
	}
	if !k.multiline && ix >= 288 && ix < 356 {
		return keyDone
	}
	return keyNone
}

func (k *touchKeyboard) pointerDown(x, y graphics.Scalar) {
	k.pressed = k.keyAt(x, y)
	k.Invalidate()
}

func (k *touchKeyboard) pointerMove(x, y graphics.Scalar) {
	key := k.keyAt(x, y)
	if key != k.pressed {
		k.pressed = key
		k.Invalidate()
	}
}

func (k *touchKeyboard) pointerLeave() {
	if k.pressed != keyNone {
		k.pressed = keyNone
		k.Invalidate()
	}
}

func keyboardCharacter(key int) string {
	characters := " !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~"
	if key < 32 || key > 126 {
		return ""
	}
	at := key - 32
	return characters[at : at+1]
}

func (k *touchKeyboard) pointerUp(x, y graphics.Scalar) {
	key := k.keyAt(x, y)
	if key == k.pressed {
		k.commitKey(key)
	}
	k.pressed = keyNone
	k.Invalidate()
}

func (k *touchKeyboard) pointerCancel() {
	if k.pressed != keyNone {
		k.pressed = keyNone
		k.Invalidate()
	}
}

func (k *touchKeyboard) commitKey(key int) {
	if key >= int('a') && key <= int('z') {
		if k.shiftMode != 0 {
			key -= int('a') - int('A')
		}
		k.append(keyboardCharacter(key))
		if k.shiftMode == 1 {
			k.shiftMode = 0
		}
	} else if key >= 32 && key <= 126 {
		k.append(keyboardCharacter(key))
	} else if key == keyShift {
		if k.symbols {
			k.symbols = false
		} else {
			k.shiftMode++
			if k.shiftMode > 2 {
				k.shiftMode = 0
			}
		}
	} else if key == keyBackspace {
		k.backspace()
	} else if key == keySymbols {
		k.symbols = !k.symbols
	} else if key == keySpace {
		k.append(" ")
	} else if key == keyEnter {
		k.append("\n")
		k.shiftMode = 1
	} else if key == keyDone {
		k.hide()
		if k.Done != nil {
			k.Done()
		}
	}
}

func (k *touchKeyboard) paintKey(surface *graphics.Surface, key int, text string, bounds graphics.Rect) {
	background := graphics.RGBA(48, 55, 67, 255)
	if key == k.pressed {
		background = graphics.RGBA(74, 134, 204, 255)
	} else if key == keyEnter || key == keyDone {
		background = graphics.RGBA(44, 145, 221, 255)
	} else if key == keyShift && k.shiftMode != 0 && !k.symbols {
		background = graphics.RGBA(75, 86, 104, 255)
	}
	surface.FillRect(bounds, background)
	surface.StrokeRect(bounds, 1, graphics.RGBA(92, 103, 120, 255))
	metrics := graphics.MeasureText(k.font, text)
	x := bounds.MinX + (bounds.Width()-metrics.Width)/2
	y := bounds.MinY + (bounds.Height()-metrics.Height)/2 + k.font.Metrics.Ascent
	surface.DrawText(k.font, graphics.Point{X: x, Y: y}, text, graphics.RGBA(240, 243, 248, 255))
}

func (k *touchKeyboard) paintRow(surface *graphics.Surface, row string, x, y, width graphics.Scalar) {
	keyWidth := width / graphics.Scalar(len(row))
	for i := 0; i < len(row); i++ {
		left := x + graphics.Scalar(i)*keyWidth
		key := int(row[i])
		label := row[i : i+1]
		if key >= int('a') && key <= int('z') && k.shiftMode != 0 {
			upper := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
			at := key - int('a')
			label = upper[at : at+1]
		}
		k.paintKey(surface, key, label, graphics.R(left+2, y+2, keyWidth-4, 68))
	}
}

func (k *touchKeyboard) paint(surface *graphics.Surface) {
	bounds := k.Bounds()
	surface.FillRect(bounds, graphics.RGBA(25, 29, 36, 255))
	surface.PushClipRect(bounds)
	surface.PushTransform()
	surface.SetTranslation(bounds.MinX, bounds.MinY)
	row0, row1, row2 := "qwertyuiop", "asdfghjkl", "zxcvbnm"
	if k.symbols {
		row0, row1, row2 = "1234567890", "@#$%&-+()", "*\"':;!?"
	}
	k.paintRow(surface, row0, 4, 4, 352)
	k.paintRow(surface, row1, 21, 80, 318)
	shiftLabel := "Shift"
	if k.symbols {
		shiftLabel = "ABC"
	} else if k.shiftMode == 2 {
		shiftLabel = "CAPS"
	}
	k.paintKey(surface, keyShift, shiftLabel, graphics.R(6, 158, 44, 68))
	k.paintRow(surface, row2, 56, 156, 248)
	k.paintKey(surface, keyBackspace, "Back", graphics.R(310, 158, 44, 68))
	symbolLabel := "?123"
	if k.symbols {
		symbolLabel = "ABC"
	}
	k.paintKey(surface, keySymbols, symbolLabel, graphics.R(6, 234, 48, 68))
	k.paintKey(surface, int(','), ",", graphics.R(62, 234, 34, 68))
	if k.multiline {
		k.paintKey(surface, keySpace, "space", graphics.R(104, 234, 96, 68))
		k.paintKey(surface, int('.'), ".", graphics.R(208, 234, 30, 68))
		k.paintKey(surface, keyEnter, "Enter", graphics.R(246, 234, 48, 68))
		k.paintKey(surface, keyDone, "Done", graphics.R(302, 234, 52, 68))
	} else {
		k.paintKey(surface, keySpace, "space", graphics.R(104, 234, 136, 68))
		k.paintKey(surface, int('.'), ".", graphics.R(248, 234, 34, 68))
		k.paintKey(surface, keyDone, "Done", graphics.R(290, 234, 64, 68))
	}
	surface.PopTransform()
	surface.PopClip()
}

var demo controlsDemo

func (d *controlsDemo) addPage(page int, control *forms.Control) {
	d.form.Add(control)
	if page == pageInputs {
		d.inputPage = append(d.inputPage, control)
	} else if page == pageLists {
		d.listPage = append(d.listPage, control)
	} else if page == pageMotion {
		d.motionPage = append(d.motionPage, control)
	} else {
		d.morePage = append(d.morePage, control)
	}
}

func setControlsVisible(controls []*forms.Control, visible bool) {
	for i := 0; i < len(controls); i++ {
		controls[i].SetVisible(visible)
	}
}

func (d *controlsDemo) showSelectedPage() {
	if d.keyboard != nil {
		d.keyboard.hide()
	}
	selected := d.tabs.SelectedIndex()
	setControlsVisible(d.inputPage, selected == pageInputs)
	setControlsVisible(d.listPage, selected == pageLists)
	setControlsVisible(d.motionPage, selected == pageMotion)
	setControlsVisible(d.morePage, selected == pageMore)
	if selected == pageInputs {
		d.status.SetText("Tap controls — every target is at least 48 dp")
	} else if selected == pageLists {
		d.status.SetText("Swipe the list to scroll, tap a row to select")
	} else if selected == pageMotion {
		d.status.SetText("Drag the slider or the split divider")
	} else {
		d.status.SetText("Toolbar, table, image and grouped controls")
	}
}

func (d *controlsDemo) showKeyboard(target *forms.Control, multiline bool) {
	d.keyboard.show(target, multiline)
	d.status.SetText("Touch keyboard active — type, backspace, then Done")
}

func (d *controlsDemo) keyboardDone() {
	d.status.SetText("Text entry complete")
}

func (d *controlsDemo) resize() {
	width, height := d.form.Size()
	if width <= 0 || height <= 0 || d.tabs == nil || d.status == nil || d.keyboard == nil {
		return
	}
	d.tabs.SetBounds(graphics.R(0, 112, graphics.Scalar(width), 52))
	statusY := graphics.Scalar(height - 56)
	d.status.SetBounds(graphics.R(0, statusY, graphics.Scalar(width), 32))
	keyboardX := graphics.Scalar(width-360) / 2
	if keyboardX < 0 {
		keyboardX = 0
	}
	d.keyboard.SetBounds(graphics.R(keyboardX, statusY-318, 360, 318))
}

func (d *controlsDemo) editTextBox() {
	if d.textBox.Text() == "Tap to focus" {
		d.textBox.SetText("")
	}
	d.showKeyboard(&d.textBox.Control, false)
}

func (d *controlsDemo) editTextArea() {
	d.showKeyboard(&d.textArea.Control, true)
}

func (d *controlsDemo) buttonClicked() {
	value := d.inputProgress.Value() + 10
	if value > 100 {
		value = 0
	}
	d.inputProgress.SetValue(value)
	d.status.SetText("Button tapped — progress advanced")
}

func (d *controlsDemo) checkChanged() {
	if d.check.Checked() {
		d.status.SetText("Checkbox enabled")
	} else {
		d.status.SetText("Checkbox disabled")
	}
}

func (d *controlsDemo) chooseRadioA() {
	d.radioA.SetChecked(true)
	d.radioB.SetChecked(false)
	d.status.SetText("Comfortable density selected")
}

func (d *controlsDemo) chooseRadioB() {
	d.radioA.SetChecked(false)
	d.radioB.SetChecked(true)
	d.status.SetText("Compact density selected")
}

func (d *controlsDemo) comboChanged() {
	d.status.SetText("Combo selection: " + d.combo.SelectedItem())
}

func (d *controlsDemo) listChanged() {
	d.status.SetText("Selected: " + d.list.SelectedItem())
}

func (d *controlsDemo) sliderChanged() {
	d.progress.SetValue(d.slider.Value())
	d.status.SetText("Slider drag is updating continuously")
}

func (d *controlsDemo) numberChanged() {
	d.slider.SetValue(d.number.Value())
	d.status.SetText("Stepper changed the slider value")
}

func (d *controlsDemo) splitChanged() {
	d.status.SetText("Split divider dragged")
}

func (d *controlsDemo) resetMotion() {
	d.number.SetValue(35)
	d.slider.SetValue(35)
	d.progress.SetValue(35)
	d.split.SetSplitterDistance(150)
	d.status.SetText("Motion controls reset")
}

func (d *controlsDemo) useDarkTheme() {
	d.form.ApplyTheme(forms.DarkTheme())
	d.status.SetText("Dark theme applied")
}

func (d *controlsDemo) useLightTheme() {
	d.form.ApplyTheme(forms.LightTheme())
	d.status.SetText("Light theme applied")
}

func (d *controlsDemo) label(page int, text string, bounds graphics.Rect, title bool) *forms.Label {
	label := forms.NewLabel()
	label.SetBounds(bounds)
	label.SetText(text)
	if title {
		label.SetFont(d.titleFont)
	} else {
		label.SetFont(d.bodyFont)
	}
	d.addPage(page, &label.Control)
	return label
}

func (d *controlsDemo) initialize(width, height int) {
	d.bodyFont = gofont.New(14)
	d.titleFont = gofont.New(20)
	d.form.Initialize(width, height)
	d.form.ApplyTheme(forms.DarkTheme())

	title := forms.NewLabel()
	title.SetBounds(graphics.R(16, 48, 328, 34))
	title.SetFont(d.titleFont)
	title.SetText("Renvo controls")
	d.form.Add(&title.Control)

	subtitle := forms.NewLabel()
	subtitle.SetBounds(graphics.R(16, 82, 328, 24))
	subtitle.SetFont(d.bodyFont)
	subtitle.SetText(controlsPlatformSubtitle)
	d.form.Add(&subtitle.Control)

	d.tabs = forms.NewTabControl()
	d.tabs.SetBounds(graphics.R(0, 112, 360, 52))
	d.tabs.SetFont(d.bodyFont)
	d.tabs.AddTab("Inputs")
	d.tabs.AddTab("Lists")
	d.tabs.AddTab("Motion")
	d.tabs.AddTab("More")
	d.tabs.Changed = d.showSelectedPage
	d.form.Add(&d.tabs.Control)

	d.status = forms.NewStatusBar()
	d.status.SetBounds(graphics.R(0, 744, 360, 32))
	d.status.SetFont(d.bodyFont)
	d.form.Add(&d.status.Control)

	d.buildInputsPage()
	d.buildListsPage()
	d.buildMotionPage()
	d.buildMorePage()
	d.keyboard = newTouchKeyboard(d.bodyFont)
	d.keyboard.SetBounds(graphics.R(0, 426, 360, 318))
	d.keyboard.Done = d.keyboardDone
	d.form.Add(&d.keyboard.Control)
	d.form.Resize = d.resize
	d.resize()
	d.showSelectedPage()
}

func (d *controlsDemo) buildInputsPage() {
	group := forms.NewGroupBox()
	group.SetBounds(graphics.R(12, 176, 336, 548))
	group.SetFont(d.bodyFont)
	group.SetText("Touch inputs")
	d.addPage(pageInputs, &group.Control)

	d.label(pageInputs, "Single-line text field", graphics.R(24, 204, 312, 24), false)
	text := forms.NewTextBox()
	text.SetBounds(graphics.R(24, 232, 312, 48))
	text.SetFont(d.bodyFont)
	text.SetText("Tap to focus")
	d.textBox = text
	text.Click = d.editTextBox
	d.addPage(pageInputs, &text.Control)

	d.label(pageInputs, "Multiline text area", graphics.R(24, 292, 312, 24), false)
	area := forms.NewTextArea()
	area.SetBounds(graphics.R(24, 320, 312, 76))
	area.SetFont(d.bodyFont)
	area.SetText("Retained controls\nrendered by Renvo")
	d.textArea = area
	area.Click = d.editTextArea
	d.addPage(pageInputs, &area.Control)

	d.check = forms.NewCheckBox()
	d.check.SetBounds(graphics.R(24, 408, 312, 48))
	d.check.SetFont(d.bodyFont)
	d.check.SetText("Enable notifications")
	d.check.SetChecked(true)
	d.check.Click = d.checkChanged
	d.addPage(pageInputs, &d.check.Control)

	d.radioA = forms.NewRadioButton()
	d.radioA.SetBounds(graphics.R(24, 460, 156, 48))
	d.radioA.SetFont(d.bodyFont)
	d.radioA.SetText("Comfortable")
	d.radioA.SetChecked(true)
	d.radioA.Click = d.chooseRadioA
	d.addPage(pageInputs, &d.radioA.Control)

	d.radioB = forms.NewRadioButton()
	d.radioB.SetBounds(graphics.R(180, 460, 156, 48))
	d.radioB.SetFont(d.bodyFont)
	d.radioB.SetText("Compact")
	d.radioB.Click = d.chooseRadioB
	d.addPage(pageInputs, &d.radioB.Control)

	button := forms.NewButton()
	button.SetBounds(graphics.R(24, 528, 312, 52))
	button.SetFont(d.bodyFont)
	button.SetText("Tap to advance progress")
	button.Click = d.buttonClicked
	d.addPage(pageInputs, &button.Control)

	d.inputProgress = forms.NewProgressBar()
	d.inputProgress.SetBounds(graphics.R(24, 596, 312, 20))
	d.inputProgress.SetValue(35)
	d.addPage(pageInputs, &d.inputProgress.Control)

	disabled := forms.NewButton()
	disabled.SetBounds(graphics.R(24, 636, 312, 52))
	disabled.SetFont(d.bodyFont)
	disabled.SetText("Disabled action")
	disabled.SetEnabled(false)
	d.addPage(pageInputs, &disabled.Control)
}

func (d *controlsDemo) buildListsPage() {
	d.label(pageLists, "Combo box", graphics.R(16, 176, 328, 24), false)
	d.combo = forms.NewComboBox()
	d.combo.SetBounds(graphics.R(16, 204, 328, 48))
	d.combo.SetFont(d.bodyFont)
	d.combo.SetMaxDropDownItems(6)
	for i := 0; i < 10; i++ {
		d.combo.AddItem([]string{"Brisbane", "Melbourne", "Sydney", "Auckland", "Tokyo", "Singapore", "London", "Berlin", "Toronto", "Seattle"}[i])
	}
	d.combo.SetSelectedIndex(0)
	d.combo.Changed = d.comboChanged
	d.addPage(pageLists, &d.combo.Control)

	d.label(pageLists, "Swipeable list", graphics.R(16, 264, 328, 24), false)
	d.list = forms.NewListBox()
	d.list.SetBounds(graphics.R(16, 292, 328, 216))
	d.list.SetFont(d.bodyFont)
	for i := 0; i < 20; i++ {
		d.list.AddItem([]string{
			"Material surface", "Buttons", "Text fields", "Check boxes", "Radio buttons",
			"Combo boxes", "List boxes", "List views", "Tree views", "Tabs",
			"Progress bars", "Numeric steppers", "Sliders", "Group boxes", "Split views",
			"Toolbars", "Status bars", "Panels", "Images", "Accessibility",
		}[i])
	}
	d.list.SetSelectedIndex(0)
	d.list.Changed = d.listChanged
	d.addPage(pageLists, &d.list.Control)

	d.label(pageLists, "Tree view", graphics.R(16, 520, 328, 24), false)
	tree := forms.NewTreeView()
	tree.SetBounds(graphics.R(16, 548, 328, 164))
	tree.SetFont(d.bodyFont)
	tree.AddNode("Controls", 0)
	tree.AddNode("Inputs", 1)
	tree.AddNode("Selection", 1)
	tree.AddNode("Layout", 1)
	tree.AddNode("Graphics", 0)
	tree.AddNode("TrueType", 1)
	tree.AddNode("Vector paths", 1)
	d.addPage(pageLists, &tree.Control)
}

func (d *controlsDemo) buildMotionPage() {
	d.label(pageMotion, "Drag slider", graphics.R(16, 184, 328, 28), true)
	d.label(pageMotion, "Pointer moves update the value continuously", graphics.R(16, 216, 328, 24), false)
	d.slider = forms.NewSlider()
	d.slider.SetBounds(graphics.R(16, 248, 328, 64))
	d.slider.SetValue(35)
	d.slider.Changed = d.sliderChanged
	d.addPage(pageMotion, &d.slider.Control)

	progress := forms.NewProgressBar()
	progress.SetBounds(graphics.R(16, 324, 328, 20))
	progress.SetValue(35)
	d.addPage(pageMotion, &progress.Control)
	d.progress = progress

	d.label(pageMotion, "Numeric stepper", graphics.R(16, 364, 160, 24), false)
	d.number = forms.NewNumericUpDown()
	d.number.SetBounds(graphics.R(184, 352, 160, 48))
	d.number.SetFont(d.bodyFont)
	d.number.SetValue(35)
	d.number.SetIncrement(5)
	d.number.Changed = d.numberChanged
	d.addPage(pageMotion, &d.number.Control)

	d.label(pageMotion, "Drag split divider", graphics.R(16, 424, 328, 24), false)
	d.split = forms.NewSplitContainer()
	d.split.SetBounds(graphics.R(16, 456, 328, 156))
	d.split.SetPanelMinimumSizes(64, 64)
	d.split.SetSplitterDistance(150)
	d.split.Changed = d.splitChanged
	d.addPage(pageMotion, &d.split.Control)

	reset := forms.NewButton()
	reset.SetBounds(graphics.R(16, 632, 328, 52))
	reset.SetFont(d.bodyFont)
	reset.SetText("Reset motion controls")
	reset.Click = d.resetMotion
	d.addPage(pageMotion, &reset.Control)
}

func (d *controlsDemo) buildMorePage() {
	toolbar := forms.NewToolBar()
	toolbar.SetBounds(graphics.R(8, 176, 344, 52))
	toolbar.SetFont(d.bodyFont)
	toolbar.AddButtonWithIcon("Dark", forms.IconSettings, d.useDarkTheme)
	toolbar.AddButtonWithIcon("Light", forms.IconRun, d.useLightTheme)
	d.addPage(pageMore, &toolbar.Control)

	listView := forms.NewListView()
	listView.SetBounds(graphics.R(16, 240, 328, 164))
	listView.SetFont(d.bodyFont)
	listView.AddColumn("Control")
	listView.AddColumn("Gesture")
	listView.SetColumnWidth(0, 160)
	listView.AddRow([]string{"Button", "Tap"})
	listView.AddRow([]string{"ListBox", "Swipe"})
	listView.AddRow([]string{"Slider", "Drag"})
	listView.AddRow([]string{"Tabs", "Tap"})
	d.addPage(pageMore, &listView.Control)

	group := forms.NewGroupBox()
	group.SetBounds(graphics.R(16, 420, 212, 128))
	group.SetFont(d.bodyFont)
	group.SetText("Group box")
	d.addPage(pageMore, &group.Control)

	groupCheck := forms.NewCheckBox()
	groupCheck.SetBounds(graphics.R(32, 452, 180, 40))
	groupCheck.SetFont(d.bodyFont)
	groupCheck.SetText("Grouped option")
	d.addPage(pageMore, &groupCheck.Control)

	groupRadio := forms.NewRadioButton()
	groupRadio.SetBounds(graphics.R(32, 496, 180, 40))
	groupRadio.SetFont(d.bodyFont)
	groupRadio.SetText("Grouped choice")
	d.addPage(pageMore, &groupRadio.Control)

	picture := forms.NewPictureBox()
	picture.SetBounds(graphics.R(244, 420, 100, 128))
	picture.SetAccessibilityName("Image placeholder")
	d.addPage(pageMore, &picture.Control)

	panel := forms.NewPanel()
	panel.SetBounds(graphics.R(16, 568, 328, 112))
	d.addPage(pageMore, &panel.Control)

	panelLabel := forms.NewLabel()
	panelLabel.SetBounds(graphics.R(32, 588, 296, 72))
	panelLabel.SetFont(d.bodyFont)
	panelLabel.SetText("Panel surface\nAll controls share one retained form")
	d.addPage(pageMore, &panelLabel.Control)
}

func main() {
	window := graphics.NewWindow(graphics.WindowOptions{
		Title: "Renvo Controls", Width: 360, Height: 800,
	})
	if window == nil {
		return
	}
	demo.initialize(360, 800)
	forms.NewApp(window, &demo.form).Run()
}
