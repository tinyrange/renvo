package graphics

import "testing"

func TestGPUCanvasTracksDamageAndBatchesAdjacentSolids(t *testing.T) {
	canvas := newGPUCanvas()
	canvas.beginFrame(200, 100, 2)
	canvas.BeginDamage(R(5, 7, 30, 20))
	canvas.PushClipRect(R(5, 7, 30, 20))
	canvas.FillRect(R(5, 7, 10, 4), redForTest())
	canvas.FillRect(R(15, 7, 10, 4), redForTest())
	canvas.PopClip()
	canvas.EndDamage()
	list, damage := canvas.endFrame()
	if len(damage) != 1 || damage[0] != (pixelRect{minX: 10, minY: 14, maxX: 70, maxY: 54}) {
		t.Fatalf("damage = %#v", damage)
	}
	if len(list.commands) != 1 {
		t.Fatalf("adjacent solid command count = %d", len(list.commands))
	}
	if len(list.vertices) != 12 || list.commands[0].count != 12 {
		t.Fatalf("batched vertices = %d, command = %#v", len(list.vertices), list.commands[0])
	}
}

func redForTest() Color { return RGBA(255, 0, 0, 255) }

func TestGPUCanvasPreservesPainterOrderAcrossStateChanges(t *testing.T) {
	canvas := newGPUCanvas()
	canvas.beginFrame(40, 40, 1)
	canvas.BeginDamage(R(0, 0, 40, 40))
	canvas.FillRect(R(0, 0, 10, 10), redForTest())
	canvas.SetBlendMode(BlendCopy)
	canvas.FillRect(R(2, 2, 10, 10), redForTest())
	canvas.SetBlendMode(BlendSourceOver)
	canvas.FillRect(R(4, 4, 10, 10), redForTest())
	canvas.EndDamage()
	list, _ := canvas.endFrame()
	if len(list.commands) != 3 {
		t.Fatalf("state-separated commands = %d", len(list.commands))
	}
	if list.commands[0].blend != BlendSourceOver || list.commands[1].blend != BlendCopy || list.commands[2].blend != BlendSourceOver {
		t.Fatalf("blend order = %v, %v, %v", list.commands[0].blend, list.commands[1].blend, list.commands[2].blend)
	}
}

func TestGPUCanvasEncodesRGBAAndMaskImages(t *testing.T) {
	canvas := newGPUCanvas()
	canvas.beginFrame(100, 100, 1)
	canvas.BeginDamage(R(0, 0, 100, 100))
	rgba := NewImage(2, 2, make([]byte, 16))
	mask := NewMask(2, 2, []byte{0, 64, 128, 255})
	canvas.DrawImage(rgba, R(0, 0, 2, 2), R(5, 6, 10, 12), SamplingLinear, White)
	canvas.DrawImage(mask, R(0, 0, 2, 2), R(20, 6, 10, 12), SamplingNearest, redForTest())
	canvas.EndDamage()
	list, _ := canvas.endFrame()
	if len(list.images) != 2 || len(list.commands) != 2 {
		t.Fatalf("images=%d commands=%d", len(list.images), len(list.commands))
	}
	if list.commands[0].kind != drawCommandRGBA || list.commands[1].kind != drawCommandMask {
		t.Fatalf("image command kinds = %v, %v", list.commands[0].kind, list.commands[1].kind)
	}
	if list.commands[0].sampling != SamplingLinear || list.commands[1].sampling != SamplingNearest {
		t.Fatalf("sampling modes lost")
	}
}

func TestGPUCanvasGeneralPathProducesClippedScanlineGeometry(t *testing.T) {
	canvas := newGPUCanvas()
	canvas.beginFrame(20, 20, 1)
	canvas.BeginDamage(R(0, 0, 20, 20))
	var path Path
	path.MoveTo(Point{2, 2})
	path.LineTo(Point{15, 2})
	path.LineTo(Point{8, 15})
	path.Close()
	canvas.FillPath(&path, FillNonZero, redForTest())
	canvas.EndDamage()
	list, _ := canvas.endFrame()
	if len(list.commands) == 0 || len(list.vertices) == 0 {
		t.Fatal("path produced no GPU geometry")
	}
	for i := 0; i < len(list.vertices); i++ {
		vertex := list.vertices[i]
		if vertex.x < 0 || vertex.x > 20 || vertex.y < 0 || vertex.y > 20 {
			t.Fatalf("unclipped vertex %d = %#v", i, vertex)
		}
	}
}

type recordingFrameBackend struct {
	began     bool
	rendered  bool
	presented bool
	commands  int
}

func (b *recordingFrameBackend) Begin(width, height int, damage []pixelRect) bool {
	b.began = width == 32 && height == 24 && len(damage) == 1
	return b.began
}

func (b *recordingFrameBackend) Render(list *drawList) bool {
	b.commands = len(list.commands)
	b.rendered = b.commands > 0
	return b.rendered
}

func (b *recordingFrameBackend) Present() bool                    { b.presented = true; return true }
func (b *recordingFrameBackend) Resize(width, height int) bool    { return true }
func (b *recordingFrameBackend) ReadPixels(surface *Surface) bool { return true }
func (b *recordingFrameBackend) Destroy()                         {}
func (b *recordingFrameBackend) Stats() RenderStats               { return RenderStats{} }

func TestAcceleratedFrameOwnsBeginRenderPresentSequence(t *testing.T) {
	window := NewWindow(WindowOptions{Width: 32, Height: 24, Hidden: true, Renderer: RendererSoftware})
	if window == nil {
		t.Fatal("headless window creation failed")
	}
	backend := &recordingFrameBackend{}
	window.setFrameBackend(RendererOpenGL, backend)
	frame := window.BeginFrame()
	canvas := frame.Canvas()
	canvas.BeginDamage(R(0, 0, 32, 24))
	canvas.FillRect(R(0, 0, 32, 24), Black)
	canvas.EndDamage()
	if !frame.Present() {
		t.Fatal("frame presentation failed")
	}
	if !backend.began || !backend.rendered || !backend.presented || backend.commands != 1 {
		t.Fatalf("backend sequence = %#v", backend)
	}
	if frame.Present() {
		t.Fatal("frame allowed double presentation")
	}
	if window.Renderer() != RendererOpenGL || window.RenderStats().Frame != 1 {
		t.Fatalf("window renderer/stats = %v %#v", window.Renderer(), window.RenderStats())
	}
}
