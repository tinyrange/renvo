//go:build (!android || !renvo) && (!renvo || !darwin || !arm64)

package forms

func paintAppWindow(app *App) bool {
	if app == nil || app.Window == nil || app.Form == nil {
		return false
	}
	if !app.Form.Paint(app.Window.Surface()) {
		return true
	}
	return app.Window.Present()
}
