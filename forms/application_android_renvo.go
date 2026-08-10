//go:build renvo && android

package forms

import "renvo.dev/std/graphics"

// App binds a compact Android Form to its NativeActivity-backed window.
type App struct {
	Window *graphics.Window
	Form   *Form
}

var androidApp *App

func NewApp(window *graphics.Window, form *Form) *App {
	return &App{Window: window, Form: form}
}

func (a *App) Run() int {
	if a == nil || a.Window == nil || a.Form == nil {
		return 1
	}
	androidApp = a
	a.Window.EventHandler = androidDispatchAppEvents
	if a.Form.Paint(a.Window.Surface()) {
		a.Window.Present()
	}
	return 0
}

func androidDispatchAppEvents() {
	a := androidApp
	if a == nil || a.Window == nil || a.Form == nil {
		return
	}
	for {
		var event graphics.Event
		if !a.Window.PollInto(&event) {
			break
		}
		a.Form.Dispatch(event)
	}
	if a.Form.Paint(a.Window.Surface()) {
		a.Window.Present()
	}
}
