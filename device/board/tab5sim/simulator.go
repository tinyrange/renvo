// Package tab5sim models the pixel traffic of the M5Stack Tab5 display path.
//
// It intentionally models the parts which are deterministic without ESP32-P4
// hardware: RGB565 framebuffer changes, retained damage, PSRAM cache writeback,
// double-buffer flips, and the continuous DSI video-mode scanout. It does not
// model CPU, cache, PSRAM, DMA, or MIPI arbitration latency.
package tab5sim

import (
	"renvo.dev/std/graphics"
)

const (
	Width           = 720
	Height          = 1280
	BytesPerPixel   = 2
	BytesPerFrame   = Width * Height * BytesPerPixel
	PixelClockHz    = 80_000_000
	HorizontalTotal = 802
	VerticalTotal   = 1544
	DSILanes        = 2
	DSILaneBitRate  = 1_040_000_000

	TargetFramesPerSec = 60
)

// RefreshHz returns the physical video-mode refresh rate implied by the Tab5
// backend's programmed pixel clock and horizontal/vertical totals.
func RefreshHz() float64 {
	return float64(PixelClockHz) / float64(HorizontalTotal*VerticalTotal)
}

// ActiveScanoutBytesPerSecond reports the RGB565 active-pixel payload consumed
// by scanout. Blanking intervals are included in the refresh-rate calculation,
// but do not fetch framebuffer pixels.
func ActiveScanoutBytesPerSecond() float64 {
	return float64(BytesPerFrame) * RefreshHz()
}

// ActiveLineBytesPerSecond is the instantaneous RGB565 payload rate while the
// controller is transmitting active pixels rather than horizontal blanking.
func ActiveLineBytesPerSecond() float64 {
	return float64(PixelClockHz * BytesPerPixel)
}

// RawLinkBytesPerSecond returns the aggregate raw capacity of both DSI lanes.
// Packet framing and D-PHY overhead consume part of this capacity.
func RawLinkBytesPerSecond() float64 {
	return float64(DSILanes*DSILaneBitRate) / 8
}

// FrameMetrics describes one buffer presented by Simulator.
type FrameMetrics struct {
	DirtyRects     int
	DamagePixels   int
	DamageBytes    int
	ChangedPixels  int
	ChangedBytes   int
	WritebackRows  int // Whole rows in the backend's coalesced damage bands.
	WritebackBytes int // Cache-line-aligned maintenance range, not measured bus traffic.
	ScanoutBytes   int
}

// Simulator owns the same two RGB565 framebuffer generations used by the
// Tab5 backend. After the initial synchronized presentation, PresentRetained
// swaps buffers without copying. Its caller must therefore repaint current and
// previous damage, just like board.PresentPortraitRetained requires.
type Simulator struct {
	frames          [2][]byte
	front           int
	back            int
	started         bool
	surface         *graphics.Surface
	framesPresented uint64
}

// New returns a zeroed Tab5 double-buffer model. The returned surface initially
// has full-frame damage, matching a newly created graphics.Surface.
func New() *Simulator {
	sim := &Simulator{front: 0, back: 1}
	sim.frames[0] = make([]byte, BytesPerFrame)
	sim.frames[1] = make([]byte, BytesPerFrame)
	sim.surface = graphics.NewSurfaceBufferFormatPreserve(
		Width, Height, graphics.PixelRGB565, sim.frames[sim.back],
	)
	return sim
}

// Surface returns the current software-rendering back buffer. Its Pixels slice
// is rebound to the old front buffer after each presentation.
func (sim *Simulator) Surface() *graphics.Surface {
	return sim.surface
}

// FramesPresented returns the number of successful simulated flips.
func (sim *Simulator) FramesPresented() uint64 {
	return sim.framesPresented
}

// ChangedPixelsOutside counts pixels for which the displayed front buffer and
// the next rendering back buffer differ outside regions. Immediately after a
// correct retained presentation, differences may remain only in the current
// frame's damage: those pixels have not yet been repainted into the old front.
func (sim *Simulator) ChangedPixelsOutside(regions []graphics.Rect) int {
	changed := 0
	front, back := sim.frames[sim.front], sim.frames[sim.back]
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			at := (y*Width + x) * BytesPerPixel
			if front[at] == back[at] && front[at+1] == back[at+1] {
				continue
			}
			inside := false
			for _, region := range regions {
				if float64(x) >= region.MinX && float64(x) < region.MaxX &&
					float64(y) >= region.MinY && float64(y) < region.MaxY {
					inside = true
					break
				}
			}
			if !inside {
				changed++
			}
		}
	}
	return changed
}

// PresentRetained measures and flips the current back buffer.
// A clean surface is a no-op, matching the board backend. The first
// presentation also synchronizes the second buffer, as the hardware backend
// does before entering its retained two-generation update loop.
func (sim *Simulator) PresentRetained() FrameMetrics {
	if sim.surface.DirtyRectCount() == 0 {
		return FrameMetrics{}
	}
	metrics := measureDamage(sim.surface)
	metrics.ScanoutBytes = BytesPerFrame
	metrics.ChangedPixels = changedPixels(sim.frames[sim.front], sim.frames[sim.back])
	metrics.ChangedBytes = metrics.ChangedPixels * BytesPerPixel

	if !sim.started {
		copy(sim.frames[sim.front], sim.frames[sim.back])
		sim.started = true
	}
	sim.front, sim.back = sim.back, sim.front
	sim.surface.Pixels = sim.frames[sim.back]
	sim.framesPresented++
	return metrics
}

func changedPixels(front, back []byte) int {
	changed := 0
	for at := 0; at+1 < len(front) && at+1 < len(back); at += BytesPerPixel {
		if front[at] != back[at] || front[at+1] != back[at+1] {
			changed++
		}
	}
	return changed
}

func measureDamage(surface *graphics.Surface) FrameMetrics {
	var metrics FrameMetrics
	if surface == nil {
		return metrics
	}
	metrics.DirtyRects = surface.DirtyRectCount()
	covered := make([]bool, Width)
	var dirtyRows [Height]bool
	for y := 0; y < Height; y++ {
		for x := range covered {
			covered[x] = false
		}
		rowDirty := false
		for region := 0; region < surface.DirtyRectCount(); region++ {
			dirty, ok := surface.DirtyRectAt(region)
			if !ok || float64(y) < dirty.MinY || float64(y) >= dirty.MaxY {
				continue
			}
			minX, maxX := int(dirty.MinX), int(dirty.MaxX)
			if minX < 0 {
				minX = 0
			}
			if maxX > Width {
				maxX = Width
			}
			if minX >= maxX {
				continue
			}
			rowDirty = true
			for x := minX; x < maxX; x++ {
				covered[x] = true
			}
		}
		if rowDirty {
			dirtyRows[y] = true
		}
		for x := range covered {
			if covered[x] {
				metrics.DamagePixels++
			}
		}
	}
	metrics.DamageBytes = metrics.DamagePixels * BytesPerPixel
	// damageBands merges adjacent bands and bands separated by one clean row.
	for y := 1; y < Height-1; y++ {
		if dirtyRows[y-1] && dirtyRows[y+1] {
			dirtyRows[y] = true
		}
	}
	for y := 0; y < Height; {
		if !dirtyRows[y] {
			y++
			continue
		}
		first := y
		for y < Height && dirtyRows[y] {
			y++
		}
		metrics.WritebackRows += y - first
		// Both PSRAM buffers are 64-byte aligned. cacheSync rounds each
		// band's start down and end up to cache-line boundaries.
		start := first * Width * BytesPerPixel / 64 * 64
		end := (y*Width*BytesPerPixel + 63) / 64 * 64
		metrics.WritebackBytes += end - start
	}
	return metrics
}
