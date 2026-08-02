package main

import (
	"renvo.dev/examples/m5sticks3/board"
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

const (
	frontButton = 11
	sideButton  = 12
	scale       = 2
	width       = board.DisplayWidth / scale
	height      = board.DisplayHeight / scale
)

func present(surface *graphics.Surface) {
	if board.PresentSurface2x(surface) {
		surface.ResetDirty()
	}
}

func waitButtonRelease(pin int) {
	for board.ButtonPressed(pin) {
		board.Delay(10000)
	}
	board.Delay(20000)
}

func main() {
	if !board.InitializeDisplay() {
		return
	}
	board.ConfigureButton(frontButton)
	board.ConfigureButton(sideButton)

	font := graphics.NewBuiltinFont(1)
	var form forms.Form
	form.Initialize(width, height)
	form.ApplyTheme(forms.DarkTheme())

	title := forms.NewLabel()
	title.SetBounds(graphics.R(3, 1, width-6, 12))
	title.SetFont(font)
	title.SetText("MENU")
	form.Add(&title.Control)

	menu := forms.NewListBox()
	menu.SetBounds(graphics.R(2, 14, width-4, height-29))
	menu.SetFont(font)
	items := []string{"Status", "Sensor", "Screen", "Radio", "Setup", "About", "Reboot", "Off"}
	for _, item := range items {
		menu.AddItem(item)
	}
	menu.SetSelectedIndex(0)
	selectedIndex := 0
	form.Add(&menu.Control)
	menu.Focus()

	result := forms.NewLabel()
	result.SetBounds(graphics.R(3, height-13, width-6, 12))
	result.SetFont(font)
	result.SetText("A: select")
	form.Add(&result.Control)

	// Build the persistent control tree before reserving the large transient
	// framebuffer so the two sides of Renvo's bounded arena cannot cross while
	// slices in the form are still growing.
	surface := graphics.NewSurface(width, height)
	form.Paint(surface)
	present(surface)
	for {
		if board.ButtonPressed(sideButton) {
			board.Delay(20000)
			if board.ButtonPressed(sideButton) {
				next := selectedIndex + 1
				if next == len(items) {
					next = 0
				}
				selectedIndex = next
				menu.SetSelectedIndex(next)
				waitButtonRelease(sideButton)
			}
		}
		if board.ButtonPressed(frontButton) {
			board.Delay(20000)
			if board.ButtonPressed(frontButton) {
				result.SetText(items[selectedIndex])
				waitButtonRelease(frontButton)
			}
		}
		if form.Paint(surface) {
			present(surface)
		}
		board.Delay(10000)
	}
}
