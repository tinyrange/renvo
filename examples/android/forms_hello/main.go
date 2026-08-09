package main

import (
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

func main() {
	window := graphics.NewWindow(graphics.WindowOptions{
		Title: "Renvo Forms", Width: 180, Height: 360,
	})
	if window == nil {
		return
	}

	var form forms.Form
	form.Initialize(180, 360)

	forms.NewApp(window, &form).Run()
}
