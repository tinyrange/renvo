//go:build renvo && browser && wasm32

package forms

import "renvo.dev/std/graphics"

var browserApp *App

func runApp(app *App) int {
	if app == nil || app.Window == nil || app.Form == nil {
		return 1
	}
	browserApp = app
	renvoBrowserStep()
	return 0
}

// renvoBrowserStep is exported by the wasm backend when present. JavaScript
// queues one browser event on stdin before calling it.
func renvoBrowserStep() {
	app := browserApp
	if app == nil || app.Window == nil {
		return
	}
	event, ok := app.Window.Wait()
	if ok {
		if event.Type == graphics.EventWindowClose {
			app.Window.Close()
			browserApp = nil
			return
		}
		app.dispatch(event)
	}
	app.paint()
}
