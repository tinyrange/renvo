//go:build renvo && ios

package forms

import "renvo.dev/std/graphics"

var iosApp *App

func runApp(app *App) int {
	if app == nil || app.Window == nil || app.Form == nil {
		return 1
	}
	iosApp = app
	app.Window.EventHandler = iosDispatchAppEvents
	if !app.paint() {
		app.Window.Close()
		return 1
	}
	return graphics.RunIOSApplication(app.Window)
}

func iosDispatchAppEvents() {
	app := iosApp
	if app == nil || app.Window == nil || app.Form == nil {
		return
	}
	for {
		var event graphics.Event
		if !app.Window.PollInto(&event) {
			break
		}
		if event.Type == graphics.EventWindowClose {
			app.Window.Close()
			return
		}
		app.dispatch(event)
	}
	app.paint()
}
