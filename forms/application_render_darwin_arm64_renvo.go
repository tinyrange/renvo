//go:build renvo && darwin && arm64

package forms

import "renvo.dev/std/graphics"

func paintAppWindow(app *App) bool {
	if app == nil || app.Window == nil || app.Form == nil {
		return false
	}
	// CAMetalLayer drawables do not preserve previous contents. Until Metal
	// renders through a private persistent target, promote a pending partial
	// repaint to a complete GPU frame. OpenGL retains exact damage in its
	// backing texture and does not take this branch.
	if app.Window.Renderer() == graphics.RendererMetal {
		width, height := app.Form.Size()
		app.Form.Invalidate(graphics.R(0, 0, graphics.Scalar(width), graphics.Scalar(height)))
	}
	frame := app.Window.BeginFrame()
	if frame == nil {
		return false
	}
	if !app.Form.Paint(frame.Canvas()) {
		frame.Cancel()
		return true
	}
	return frame.Present()
}
