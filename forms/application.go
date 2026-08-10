//go:build !android || !renvo

package forms

import "renvo.dev/std/graphics"

// App gives the host control of the event loop while keeping application code
// identical across native and browser targets.
type App struct {
	Window        *graphics.Window
	Form          *Form
	DispatchEvent func(graphics.Event)
	AfterEvent    func(graphics.Event)
	BeforeRender  func()
	AfterRender   func()
}

func NewApp(window *graphics.Window, form *Form) *App {
	return &App{Window: window, Form: form}
}

func (a *App) Run() int { return runApp(a) }

func (a *App) paint() bool {
	if a == nil || a.Window == nil || a.Form == nil {
		return false
	}
	if len(a.Form.invalid) == 0 {
		return true
	}
	if a.BeforeRender != nil {
		a.BeforeRender()
	}
	if a.Form.Paint(a.Window.Surface()) {
		if !a.Window.Present() {
			return false
		}
		if a.AfterRender != nil {
			a.AfterRender()
		}
	}
	return true
}

func (a *App) dispatch(event graphics.Event) {
	if a == nil || a.Form == nil {
		return
	}
	if a.DispatchEvent != nil {
		a.DispatchEvent(event)
	} else {
		a.Form.Dispatch(event)
	}
	if a.AfterEvent != nil {
		a.AfterEvent(event)
	}
	if a.Window != nil && (event.Type == graphics.EventPointerMove || event.Type == graphics.EventPointerDown || event.Type == graphics.EventPointerUp) {
		a.Window.SetCursor(a.Form.CursorAt(event.X, event.Y))
	} else if a.Window != nil && event.Type == graphics.EventPointerLeave {
		a.Window.SetCursor(graphics.CursorArrow)
	}
}
