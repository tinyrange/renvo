//go:build !browser

package forms

import "renvo.dev/std/graphics"

func runApp(app *App) int {
	if app == nil || app.Window == nil || app.Form == nil {
		return 1
	}
	for {
		if !app.paint() {
			app.Window.Close()
			return 1
		}
		event, ok := app.Window.Wait()
		if !ok || event.Type == graphics.EventWindowClose {
			app.Window.Close()
			return 0
		}
		app.dispatch(event)
	}
}
