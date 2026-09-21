//go:build m5tab5

package main

import (
	"sort"
	"testing"
	"time"

	"renvo.dev/device/board/tab5sim"
	"renvo.dev/examples/device/fontcache"
	"renvo.dev/std/graphics"
)

const tab5FrameBudget = time.Second / tab5sim.TargetFramesPerSec

type tab5AutomationResult struct {
	frames         int
	totalDamage    int
	totalChanged   int
	totalWriteback int
	peakDamage     int
	peakChanged    int
	peakWriteback  int
	paintTimes     []time.Duration
}

func captureInvalidRects(demo *controlsDemo) []graphics.Rect {
	count := demo.form.InvalidRectCount()
	regions := make([]graphics.Rect, 0, count)
	for index := 0; index < count; index++ {
		region, ok := demo.form.InvalidRectAt(index)
		if ok {
			regions = append(regions, region)
		}
	}
	return regions
}

func paintRetainedFrame(demo *controlsDemo, sim *tab5sim.Simulator, previous []graphics.Rect) ([]graphics.Rect, tab5sim.FrameMetrics, time.Duration, int) {
	current := captureInvalidRects(demo)
	for _, region := range previous {
		demo.form.Invalidate(region)
	}
	started := time.Now()
	demo.form.Paint(sim.Surface())
	paintTime := time.Since(started)
	demo.overlay.afterPaint()
	metrics := sim.PresentRetained()
	outside := sim.ChangedPixelsOutside(current)
	sim.Surface().ResetDirty()
	return current, metrics, paintTime, outside
}

func runTab5SliderAutomation(t testing.TB, frames int) tab5AutomationResult {
	t.Helper()
	font, titleFont := fontcache.Body(), fontcache.Title()
	if font == nil || titleFont == nil {
		t.Fatal("could not load the Tab5 cached fonts")
	}

	var demo controlsDemo
	demo.initialize(font, titleFont)
	sim := tab5sim.New()
	if !demo.form.Paint(sim.Surface()) {
		t.Fatal("initial Forms paint reported no work")
	}
	demo.overlay.afterPaint()
	sim.PresentRetained()
	sim.Surface().ResetDirty()

	result := tab5AutomationResult{paintTimes: make([]time.Duration, 0, frames)}
	previous := []graphics.Rect(nil)
	bounds := demo.slider.Bounds()
	startX := bounds.MinX + 8 + (bounds.Width()-16)*graphics.Scalar(demo.slider.Value())/100
	endX := bounds.MaxX - 8
	y := bounds.MinY + bounds.Height()/2

	for frame := 0; frame < frames; frame++ {
		x := startX
		if frames > 1 {
			x += (endX - startX) * graphics.Scalar(frame) / graphics.Scalar(frames-1)
		}
		eventType := graphics.EventPointerMove
		if frame == 0 {
			eventType = graphics.EventPointerDown
		} else if frame == frames-1 {
			eventType = graphics.EventPointerUp
		}
		demo.form.Dispatch(graphics.Event{Type: eventType, X: x, Y: y, Button: 1})
		if demo.form.InvalidRectCount() == 0 {
			t.Fatalf("automation frame %d produced no Forms damage", frame)
		}

		var metrics tab5sim.FrameMetrics
		var paintTime time.Duration
		var outside int
		previous, metrics, paintTime, outside = paintRetainedFrame(&demo, sim, previous)
		if outside != 0 {
			t.Fatalf("automation frame %d left %d stale pixels outside current damage", frame, outside)
		}
		result.frames++
		result.paintTimes = append(result.paintTimes, paintTime)
		result.totalDamage += metrics.DamageBytes
		result.totalChanged += metrics.ChangedBytes
		result.totalWriteback += metrics.WritebackBytes
		if metrics.DamageBytes > result.peakDamage {
			result.peakDamage = metrics.DamageBytes
		}
		if metrics.ChangedBytes > result.peakChanged {
			result.peakChanged = metrics.ChangedBytes
		}
		if metrics.WritebackBytes > result.peakWriteback {
			result.peakWriteback = metrics.WritebackBytes
		}
	}
	if got := sim.FramesPresented(); got != uint64(frames+1) {
		t.Fatalf("simulated flips = %d, want %d including initial paint", got, frames+1)
	}
	if demo.slider.Value() != 100 {
		t.Fatalf("automated slider value = %d, want 100", demo.slider.Value())
	}
	return result
}

func percentile(values []time.Duration, numerator, denominator int) time.Duration {
	ordered := append([]time.Duration(nil), values...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i] < ordered[j] })
	index := (len(ordered)*numerator + denominator - 1) / denominator
	if index > 0 {
		index--
	}
	return ordered[index]
}

func TestTab5SliderDragAt60FPS(t *testing.T) {
	const frames = tab5sim.TargetFramesPerSec
	result := runTab5SliderAutomation(t, frames)
	p50 := percentile(result.paintTimes, 50, 100)
	p95 := percentile(result.paintTimes, 95, 100)
	maximum := percentile(result.paintTimes, 100, 100)

	if result.frames != frames {
		t.Fatalf("presented frames = %d, want %d", result.frames, frames)
	}
	if p95 > tab5FrameBudget {
		t.Fatalf("host-simulated p95 paint = %s, exceeds 60 FPS budget %s", p95, tab5FrameBudget)
	}
	t.Logf("Tab5 timing: %.2f Hz scanout, %.3f MB/frame, %.3f MB/s active RGB565 payload",
		tab5sim.RefreshHz(), float64(tab5sim.BytesPerFrame)/1e6, tab5sim.ActiveScanoutBytesPerSecond()/1e6)
	t.Logf("active-line payload=%.1f MB/s; two-lane DSI raw capacity=%.1f MB/s",
		tab5sim.ActiveLineBytesPerSecond()/1e6, tab5sim.RawLinkBytesPerSecond()/1e6)
	t.Logf("60-frame slider drag: paint p50=%s p95=%s max=%s (budget=%s)", p50, p95, maximum, tab5FrameBudget)
	t.Logf("per UI frame: damage avg=%.1f KB peak=%.1f KB; changed avg=%.1f KB peak=%.1f KB; cache writeback avg=%.1f KB peak=%.1f KB",
		float64(result.totalDamage)/float64(frames)/1e3, float64(result.peakDamage)/1e3,
		float64(result.totalChanged)/float64(frames)/1e3, float64(result.peakChanged)/1e3,
		float64(result.totalWriteback)/float64(frames)/1e3, float64(result.peakWriteback)/1e3)
}

func BenchmarkTab5SliderDrag60Frames(b *testing.B) {
	for iteration := 0; iteration < b.N; iteration++ {
		result := runTab5SliderAutomation(b, tab5sim.TargetFramesPerSec)
		var total time.Duration
		for _, elapsed := range result.paintTimes {
			total += elapsed
		}
		b.ReportMetric(float64(total.Nanoseconds())/float64(result.frames), "paint-ns/frame")
		b.ReportMetric(float64(result.totalChanged)/float64(result.frames), "changed-B/frame")
		b.ReportMetric(float64(result.totalDamage)/float64(result.frames), "damage-B/frame")
		b.ReportMetric(float64(result.totalWriteback)/float64(result.frames), "writeback-B/frame")
	}
}
