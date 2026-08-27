package forms_test

import (
	"fmt"

	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

// Forms events are ordinary callbacks and can capture application state.
func ExampleButton() {
	button := forms.NewButton()
	button.SetText("Deploy")
	clicks := 0
	button.Click = func() { clicks++ }
	button.Click()

	fmt.Println(button.Text(), clicks)
	// Output: Deploy 1
}

func ExampleNewButton() {
	button := forms.NewButton()
	button.SetText("Save")
	button.Click = func() { /* save the document */ }
}

func ExampleNewCheckBox() {
	checkBox := forms.NewCheckBox()
	checkBox.SetText("Enable telemetry")
	checkBox.SetChecked(true)
}

func ExampleNewComboBox() {
	combo := forms.NewComboBox()
	combo.AddItem("Debug")
	combo.AddItem("Release")
	combo.SetSelectedIndex(1)
}

func ExampleNewGroupBox() {
	group := forms.NewGroupBox()
	group.SetText("Connection")
	group.SetBounds(graphics.R(16, 16, 280, 120))
}

func ExampleNewLabel() {
	label := forms.NewLabel()
	label.SetText("Device ready")
	label.SetForeground(graphics.RGBA(40, 160, 80, 255))
}

func ExampleNewListBox() {
	list := forms.NewListBox()
	list.AddItem("COM1")
	list.AddItem("COM2")
	list.SetSelectedIndex(0)
}

func ExampleNewListView() {
	list := forms.NewListView()
	list.AddColumn("Name")
	list.AddColumn("Size")
	list.AddRow([]string{"kernel.efi", "96 KiB"})
}

func ExampleNewMenu() {
	file := forms.NewMenu("File")
	file.Add(forms.NewMenuItem("Open"))
	file.Add(forms.NewMenuItem("Save"))
}

func ExampleNewMenuBar() {
	bar := forms.NewMenuBar()
	bar.Add(forms.NewMenu("File"))
	bar.Add(forms.NewMenu("Help"))
}

func ExampleNewMenuItem() {
	item := forms.NewMenuItem("Build")
	item.SetIcon(forms.IconBuild)
	item.Activate = func() { /* build the project */ }
}

func ExampleNewNumericUpDown() {
	number := forms.NewNumericUpDown()
	number.SetRange(1, 100)
	number.SetIncrement(5)
	number.SetValue(25)
}

func ExampleNewPanel() {
	panel := forms.NewPanel()
	panel.SetDock(forms.DockFill)
	panel.SetBackground(graphics.RGBA(32, 36, 44, 255))
}

func ExampleNewPictureBox() {
	picture := forms.NewPictureBox()
	picture.SetBounds(graphics.R(8, 8, 128, 128))
	picture.SetAccessibilityName("Board preview")
}

func ExampleNewProgressBar() {
	progress := forms.NewProgressBar()
	progress.SetRange(0, 100)
	progress.SetValue(65)
}

func ExampleNewRadioButton() {
	radio := forms.NewRadioButton()
	radio.SetText("Use native backend")
	radio.SetChecked(true)
}

func ExampleNewSlider() {
	volume := forms.NewSlider()
	volume.SetRange(0, 100)
	volume.SetSmallChange(1)
	volume.SetLargeChange(10)
	volume.SetValue(75)
}

func ExampleNewSplitContainer() {
	split := forms.NewSplitContainer()
	split.SetVertical(true)
	split.SetPanelMinimumSizes(180, 240)
	split.SetSplitterDistance(320)
}

func ExampleNewStatusBar() {
	status := forms.NewStatusBar()
	status.SetText("Ready")
	status.SetDock(forms.DockBottom)
}

func ExampleNewTabControl() {
	tabs := forms.NewTabControl()
	tabs.AddTabWithIcon("Code", forms.IconCode)
	tabs.AddTabWithIcon("Designer", forms.IconDesigner)
}

func ExampleNewTextArea() {
	editor := forms.NewTextArea()
	editor.SetText("package main\n\nfunc main() {}\n")
	editor.SetDock(forms.DockFill)
}

func ExampleNewTextBox() {
	name := forms.NewTextBox()
	name.SetText("firmware")
	name.SetAccessibilityName("Output name")
}

func ExampleNewToolBar() {
	toolbar := forms.NewToolBar()
	toolbar.AddButtonWithIcon("Build", forms.IconBuild, func() { /* build */ })
	toolbar.AddButtonWithIcon("Run", forms.IconRun, func() { /* run */ })
}

func ExampleNewTreeView() {
	tree := forms.NewTreeView()
	tree.AddNode("device", 0)
	tree.AddNode("gpio", 1)
	tree.AddNode("i2c", 1)
	tree.ExpandAll()
}

func ExampleNewControl() {
	control := forms.NewControl()
	control.SetBounds(graphics.R(10, 10, 120, 32))
	control.SetCursor(graphics.CursorPointingHand)
}

func ExampleDarkTheme() {
	theme := forms.DarkTheme()
	var form forms.Form
	form.ApplyTheme(theme)
}

func ExampleLightTheme() {
	theme := forms.LightTheme()
	button := forms.NewButton()
	button.ApplyTheme(theme)
}

func ExampleIconName() {
	for icon := forms.Icon(0); int(icon) < forms.IconCount(); icon++ {
		name := forms.IconName(icon)
		_ = name
	}
}

func ExampleForm_Initialize() {
	var form forms.Form
	form.Initialize(640, 480)
	width, height := form.Size()
	_, _ = width, height
}

func ExampleForm_Add() {
	var form forms.Form
	form.Initialize(640, 480)
	button := forms.NewButton()
	button.SetText("Compile")
	form.Add(&button.Control)
}

func ExampleForm_ApplyTheme() {
	var form forms.Form
	form.Initialize(640, 480)
	form.ApplyTheme(forms.DarkTheme())
}

func ExampleControl_SetBounds() {
	button := forms.NewButton()
	button.SetBounds(graphics.R(20, 20, 120, 36))
}

func ExampleControl_SetDock() {
	toolbar := forms.NewToolBar()
	toolbar.SetDock(forms.DockTop)
	status := forms.NewStatusBar()
	status.SetDock(forms.DockBottom)
}

func ExampleControl_SetAccessibilityName() {
	button := forms.NewButton()
	button.SetText("▶")
	button.SetAccessibilityName("Run program")
	button.SetAccessibilityDescription("Compiles and runs the current project")
}

func ExampleControl_BeginTextEdit() {
	field := forms.NewTextBox()
	field.BeginTextEdit(64)
	field.AppendText("renvo")
	field.EndTextEdit(true)
}

func ExampleComboBox_SetSelectedIndex() {
	combo := forms.NewComboBox()
	combo.AddItem("BIOS 8086")
	combo.AddItem("UEFI AMD64")
	combo.SetSelectedIndex(1)
}

func ExampleListView_AddRow() {
	list := forms.NewListView()
	list.AddColumn("Target")
	list.AddColumn("Status")
	list.AddRow([]string{"wasi/wasm32", "Runnable"})
	list.AddRow([]string{"esp32c6/riscv32", "Flashable"})
}

func ExampleMenuItem_SetShortcut() {
	save := forms.NewMenuItem("Save")
	save.SetShortcut(forms.Shortcut{
		Key:       graphics.KeyS,
		Modifiers: graphics.ModifierControl,
		Primary:   true,
		Text:      "Ctrl+S",
	})
}

func ExampleAutomationDriver_Invoke() {
	var form forms.Form
	form.Initialize(320, 200)
	button := forms.NewButton()
	button.SetText("Deploy")
	button.SetAccessibilityID("deploy")
	form.Add(&button.Control)

	driver := forms.NewAutomationDriver(&form)
	driver.Invoke("deploy")
}

func ExampleTreeView_SetExpanded() {
	tree := forms.NewTreeView()
	tree.AddNode("device", 0)
	tree.AddNode("sensor", 1)
	tree.SetExpanded(0, true)
}

func ExampleTabControl_AddTabWithIcon() {
	tabs := forms.NewTabControl()
	tabs.AddTabWithIcon("Project", forms.IconFolder)
	tabs.AddTabWithIcon("Search", forms.IconSearch)
}
