package main

import (
	board "renvo.dev/device/board/m5sticks3"
	"renvo.dev/device/gpio"
	"renvo.dev/forms"
	"renvo.dev/std/graphics"
)

const (
	scale  = 2
	width  = 135 / scale
	height = 240 / scale
)

func present(surface *graphics.Surface) {
	if board.Display.PresentSurface2x(surface) {
		surface.ResetDirty()
	}
}

func waitButtonRelease(button *gpio.Button) {
	for button.Pressed() {
		board.Clock.DelayMilliseconds(10)
	}
	board.Clock.DelayMilliseconds(20)
}

func main() {
	if !board.Display.Initialize() {
		return
	}

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
		if board.SideButton.Pressed() {
			board.Clock.DelayMilliseconds(20)
			if board.SideButton.Pressed() {
				next := selectedIndex + 1
				if next == len(items) {
					next = 0
				}
				selectedIndex = next
				menu.SetSelectedIndex(next)
				waitButtonRelease(&board.SideButton)
			}
		}
		if board.FrontButton.Pressed() {
			board.Clock.DelayMilliseconds(20)
			if board.FrontButton.Pressed() {
				result.SetText(items[selectedIndex])
				waitButtonRelease(&board.FrontButton)
			}
		}
		if form.Paint(surface) {
			present(surface)
		}
		board.Clock.DelayMilliseconds(10)
	}
}
