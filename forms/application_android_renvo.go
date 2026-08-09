//go:build renvo && android

package forms

import "renvo.dev/std/graphics"

// App binds a compact Android Form to its NativeActivity-backed window.
type App struct {
	Window *graphics.Window
	Form   *Form
}

func NewApp(window *graphics.Window, form *Form) *App {
	return &App{Window: window, Form: form}
}

func (a *App) Run() int {
	if a == nil || a.Window == nil || a.Form == nil {
		return 1
	}
	if a.Form.Paint(a.Window.Surface()) {
		a.Window.Present()
	}
	return 0
}
