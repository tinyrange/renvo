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

func paintAndroidApp(a *App) bool {
	if a == nil || a.Window == nil || a.Form == nil {
		return false
	}
	// EGL swaps do not preserve partial Forms damage without a persistent target.
	if a.Window.Renderer() == graphics.RendererOpenGL {
		width, height := a.Form.Size()
		a.Form.Invalidate(graphics.R(0, 0, graphics.Scalar(width), graphics.Scalar(height)))
	}
	frame := a.Window.BeginFrame()
	if frame == nil {
		return false
	}
	if !a.Form.Paint(frame.Canvas()) {
		frame.Cancel()
		return true
	}
	return frame.Present()
}

func (a *App) Run() int {
	if a == nil || a.Window == nil || a.Form == nil {
		return 1
	}
	androidApp = a
	a.Window.EventHandler = androidDispatchAppEvents
	paintAndroidApp(a)
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
	paintAndroidApp(a)
}
