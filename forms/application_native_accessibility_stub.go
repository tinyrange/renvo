//go:build !browser && (!renvo || !darwin || !arm64)

package forms

func syncNativeAccessibility(app *App) {}
