package main

import (
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
	"renvo.dev/std/graphics/gofont"
)

var demoForm forms.Form
var demoBody *forms.Label
var demoButton *forms.Button

func continueClicked() {
	demoBody.SetText("Touch input is responsive")
	demoButton.SetText("Working")
}

func main() {
	window := graphics.NewWindow(graphics.WindowOptions{
		Title: "Renvo Forms", Width: 360, Height: 800,
	})
	if window == nil {
		return
	}

	font := gofont.New(16)
	titleFont := gofont.New(24)
	demoForm.Initialize(360, 800)
	demoForm.ApplyTheme(forms.DarkTheme())

	title := forms.NewLabel()
	title.SetBounds(graphics.R(24, 72, 312, 48))
	title.SetFont(titleFont)
	title.SetText(formsHelloTitle)
	demoForm.Add(&title.Control)

	demoBody = forms.NewLabel()
	demoBody.SetBounds(graphics.R(24, 128, 312, 48))
	demoBody.SetFont(font)
	demoBody.SetText("Full TrueType text")
	demoForm.Add(&demoBody.Control)

	demoButton = forms.NewButton()
	demoButton.SetBounds(graphics.R(24, 192, 312, 52))
	demoButton.SetFont(font)
	demoButton.SetText("Continue")
	demoButton.Click = continueClicked
	demoForm.Add(&demoButton.Control)

	forms.NewApp(window, &demoForm).Run()
}
