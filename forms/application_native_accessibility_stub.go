//go:build !browser && (!renvo || !darwin || !arm64 || ios)

package forms

func syncNativeAccessibility(app *App) {}
