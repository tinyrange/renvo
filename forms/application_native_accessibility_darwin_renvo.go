//go:build renvo && darwin && arm64

package forms

func syncNativeAccessibility(app *App) {
	if app == nil || app.Window == nil || app.Form == nil {
		return
	}
	update, ok := app.Form.TakeAccessibilityUpdate()
	if ok {
		app.Window.PresentAccessibility(encodeAccessibilityUpdate(update))
	}
}
