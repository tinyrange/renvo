package main

import (
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
	"renvo.dev/std/graphics/gofont"
)

func main() {
	font := gofont.New(15)
	if font == nil {
		print("FAIL font nil\n")
		return
	}
	if graphics.MeasureText(font, "WW").Width <= graphics.MeasureText(font, "ii").Width {
		print("FAIL font\n")
		return
	}
	var form forms.Form
	form.Initialize(360, 260)
	panel := forms.NewPanel()
	panel.SetBounds(graphics.R(8, 8, 344, 244))
	label := forms.NewLabel()
	label.SetBounds(graphics.R(20, 20, 180, 28))
	label.SetFont(font)
	label.SetText("Embedded font")
	button := forms.NewButton()
	button.SetBounds(graphics.R(210, 18, 120, 34))
	button.SetFont(font)
	button.SetText("Button")
	textBox := forms.NewTextBox()
	textBox.SetBounds(graphics.R(20, 62, 150, 32))
	textBox.SetFont(font)
	textBox.SetText("Text input")
	textArea := forms.NewTextArea()
	textArea.SetBounds(graphics.R(180, 62, 150, 70))
	textArea.SetFont(font)
	textArea.SetText("Text\narea")
	checkBox := forms.NewCheckBox()
	checkBox.SetBounds(graphics.R(20, 108, 140, 28))
	checkBox.SetFont(font)
	checkBox.SetText("Checked")
	checkBox.SetChecked(true)
	radio := forms.NewRadioButton()
	radio.SetBounds(graphics.R(20, 144, 140, 28))
	radio.SetFont(font)
	radio.SetText("Selected")
	radio.SetChecked(true)
	picture := forms.NewPictureBox()
	picture.SetBounds(graphics.R(180, 144, 150, 82))
	form.Add(&panel.Control)
	form.Add(&label.Control)
	form.Add(&button.Control)
	form.Add(&textBox.Control)
	form.Add(&textArea.Control)
	form.Add(&checkBox.Control)
	form.Add(&radio.Control)
	form.Add(&picture.Control)
	if !checkBox.Checked() || !radio.Checked() || !form.Paint(graphics.NewSurface(360, 260)) {
		print("FAIL controls\n")
		return
	}
	print("PASS\n")
}
