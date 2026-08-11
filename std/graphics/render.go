package graphics

// Renderer selects the implementation used by window frames. Surface and
// Window.Present remain the explicit software framebuffer API; Forms uses
// BeginFrame so RendererAuto may choose an accelerated canvas where supported.
type Renderer int

const (
	RendererAuto Renderer = iota
	RendererSoftware
	RendererMetal
	RendererOpenGL
)

// RenderStats describes the most recently presented frame. GPUSeconds is zero
// when a backend cannot provide a trustworthy GPU timestamp.
type RenderStats struct {
	Renderer       Renderer
	Frame          int
	DamageRects    int
	Commands       int
	Triangles      int
	TextureBytes   int
	EncodeSeconds  Scalar
	SubmitSeconds  Scalar
	GPUSeconds     Scalar
	PresentSeconds Scalar
}

type frameBackend interface {
	Begin(width, height int, damage []pixelRect) bool
	Render(list *drawList) bool
	Present() bool
	Resize(width, height int) bool
	ReadPixels(surface *Surface) bool
	Destroy()
	Stats() RenderStats
}

// Frame owns one balanced window paint. Its Canvas is either the window's
// software Surface or a GPU command encoder; callers need not branch on it.
type Frame struct {
	window      *Window
	canvas      Canvas
	accelerated bool
	ended       bool
}

func (w *Window) BeginFrame() *Frame {
	if w == nil || w.closed {
		return nil
	}
	if w.backend != nil && w.frameCanvas != nil {
		w.frameCanvas.beginFrame(w.surface.Width, w.surface.Height, w.surface.deviceScale)
		return &Frame{window: w, canvas: w.frameCanvas, accelerated: true}
	}
	return &Frame{window: w, canvas: w.surface}
}

func (f *Frame) Canvas() Canvas {
	if f == nil || f.ended {
		return nil
	}
	return f.canvas
}

func (f *Frame) Present() bool {
	if f == nil || f.ended || f.window == nil {
		return false
	}
	f.ended = true
	if !f.accelerated {
		return f.window.Present()
	}
	w := f.window
	list, damage := w.frameCanvas.endFrame()
	if len(damage) == 0 {
		return true
	}
	if !w.backend.Begin(w.surface.Width, w.surface.Height, damage) {
		return false
	}
	if !w.backend.Render(list) {
		return false
	}
	if !w.backend.Present() {
		return false
	}
	stats := w.backend.Stats()
	stats.Renderer = w.renderer
	stats.Frame = w.renderStats.Frame + 1
	stats.DamageRects = len(damage)
	stats.Commands = len(list.commands)
	stats.Triangles = len(list.vertices) / 3
	w.renderStats = stats
	return true
}

func (f *Frame) Cancel() {
	if f == nil || f.ended {
		return
	}
	f.ended = true
	if f.accelerated && f.window != nil && f.window.frameCanvas != nil {
		f.window.frameCanvas.cancelFrame()
	}
}

func (w *Window) Renderer() Renderer {
	if w == nil || w.backend == nil {
		return RendererSoftware
	}
	return w.renderer
}

func (w *Window) RenderStats() RenderStats {
	if w == nil {
		return RenderStats{}
	}
	return w.renderStats
}

func (w *Window) setFrameBackend(renderer Renderer, backend frameBackend) {
	if w == nil || backend == nil {
		return
	}
	w.renderer = renderer
	w.backend = backend
	w.frameCanvas = newGPUCanvas()
}
