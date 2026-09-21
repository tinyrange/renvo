package tab5sim

import (
	"math"
	"testing"

	"renvo.dev/std/graphics"
)

func TestVideoTimingAndBandwidth(t *testing.T) {
	if BytesPerFrame != 1_843_200 {
		t.Fatalf("bytes per frame = %d, want 1843200", BytesPerFrame)
	}
	if got := RefreshHz(); math.Abs(got-64.61) > 0.01 {
		t.Fatalf("refresh = %.4f Hz, want about 64.61 Hz", got)
	}
	if got := ActiveScanoutBytesPerSecond(); math.Abs(got-119_090_000) > 20_000 {
		t.Fatalf("active scanout = %.0f B/s, want about 119.09 MB/s", got)
	}
	if ActiveLineBytesPerSecond() != 160_000_000 {
		t.Fatalf("active-line payload = %.0f B/s, want 160 MB/s", ActiveLineBytesPerSecond())
	}
	if RawLinkBytesPerSecond() != 260_000_000 {
		t.Fatalf("raw link capacity = %.0f B/s, want 260 MB/s", RawLinkBytesPerSecond())
	}
	if RefreshHz() < TargetFramesPerSec {
		t.Fatalf("refresh %.2f Hz cannot sustain %d FPS", RefreshHz(), TargetFramesPerSec)
	}
}

func TestRetainedFlipMeasuresPixelTraffic(t *testing.T) {
	sim := New()
	surface := sim.Surface()
	surface.FillRect(graphics.R(0, 0, Width, Height), graphics.RGBA(0x10, 0x20, 0x30, 0xff))
	first := sim.PresentRetained()
	if first.DamagePixels != Width*Height || first.WritebackBytes != BytesPerFrame {
		t.Fatalf("initial traffic = %#v", first)
	}
	surface.ResetDirty()

	// Two overlapping rectangles cover 16 unique pixels over four dirty rows.
	surface.FillRect(graphics.R(10, 20, 3, 3), graphics.RGBA(0xff, 0, 0, 0xff))
	surface.FillRect(graphics.R(12, 21, 3, 3), graphics.RGBA(0, 0xff, 0, 0xff))
	frame := sim.PresentRetained()
	if frame.DirtyRects != 2 {
		t.Fatalf("dirty rects = %d, want 2", frame.DirtyRects)
	}
	if frame.DamagePixels != 16 {
		t.Fatalf("damage pixels = %d, want union area 16", frame.DamagePixels)
	}
	if frame.WritebackRows != 4 || frame.WritebackBytes != 4*Width*BytesPerPixel {
		t.Fatalf("writeback = %d rows, %d bytes", frame.WritebackRows, frame.WritebackBytes)
	}
	if frame.ChangedPixels != 16 {
		t.Fatalf("changed pixels = %d, want 16", frame.ChangedPixels)
	}
	if frame.ScanoutBytes != BytesPerFrame {
		t.Fatalf("scanout bytes = %d, want %d", frame.ScanoutBytes, BytesPerFrame)
	}
	if outside := sim.ChangedPixelsOutside([]graphics.Rect{graphics.R(10, 20, 5, 4)}); outside != 0 {
		t.Fatalf("changed pixels outside current damage = %d, want 0", outside)
	}
}

func TestCleanPresentationDoesNotFlip(t *testing.T) {
	sim := New()
	sim.PresentRetained()
	surface := sim.Surface()
	surface.ResetDirty()
	surface.FillRect(graphics.R(10, 20, 1, 1), graphics.RGBA(255, 0, 0, 255))
	sim.PresentRetained()
	surface.ResetDirty()
	back := &surface.Pixels[0]
	if metrics := sim.PresentRetained(); metrics != (FrameMetrics{}) {
		t.Fatalf("clean presentation traffic = %#v", metrics)
	}
	if sim.FramesPresented() != 2 || &surface.Pixels[0] != back {
		t.Fatal("clean presentation flipped the stale back buffer")
	}
}

func TestWritebackMatchesCoalescedAlignedBands(t *testing.T) {
	for _, test := range []struct {
		name                string
		rows                []int
		wantRows, wantBytes int
	}{
		{"even row", []int{20}, 1, 1472},
		{"odd row", []int{21}, 1, 1472},
		{"one row gap", []int{20, 22}, 3, 4352},
		{"separate bands", []int{20, 23}, 2, 2944},
	} {
		t.Run(test.name, func(t *testing.T) {
			sim := New()
			surface := sim.Surface()
			surface.ResetDirty()
			for _, y := range test.rows {
				surface.FillRect(graphics.R(10, graphics.Scalar(y), 1, 1), graphics.RGBA(255, 0, 0, 255))
			}
			metrics := sim.PresentRetained()
			if metrics.DamagePixels != len(test.rows) || metrics.WritebackRows != test.wantRows || metrics.WritebackBytes != test.wantBytes {
				t.Fatalf("traffic = %#v, want %d damaged pixels, %d writeback rows, %d writeback bytes", metrics, len(test.rows), test.wantRows, test.wantBytes)
			}
		})
	}
}
