package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/display/ssd1677"
	"renvo.dev/device/input/ft6336g"
)

const (
	trailThickness        = 2
	presentIntervalMillis = uint32(10)
	logIntervalMillis     = uint32(20)
	maximumQueuedTouches  = 64
)

var frame ssd1677.Monochrome

type touchEvent struct {
	x, y    int
	pressed bool
}

type trails struct {
	events              [maximumQueuedTouches]touchEvent
	count               int
	sampledX, sampledY  int
	sampledPressed      bool
	drawnPressed, dirty bool
	previous            ft6336g.Point
	lastLog             uint32
}

func fail(message string) {
	print(message)
	_ = board.Display.Shutdown()
	for {
	}
}

// The official PaperMono orientation is M5GFX rotation zero with panel offset
// rotation three: portrait (x,y) maps to native framebuffer (y,479-x).
func setPortraitPixel(x, y int, white bool) {
	frame.Set(y, ssd1677.Height-1-x, white)
}

func absolute(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func drawBrush(x, y int) {
	for offsetY := 0; offsetY < trailThickness; offsetY++ {
		for offsetX := 0; offsetX < trailThickness; offsetX++ {
			setPortraitPixel(x+offsetX, y+offsetY, false)
		}
	}
}

func drawTrail(from, to ft6336g.Point) {
	x, y := from.X, from.Y
	deltaX := absolute(to.X - x)
	stepX := -1
	if x < to.X {
		stepX = 1
	}
	deltaY := -absolute(to.Y - y)
	stepY := -1
	if y < to.Y {
		stepY = 1
	}
	err := deltaX + deltaY
	for {
		drawBrush(x, y)
		if x == to.X && y == to.Y {
			return
		}
		twice := err * 2
		if twice >= deltaY {
			err += deltaY
			x += stepX
		}
		if twice <= deltaX {
			err += deltaX
			y += stepY
		}
	}
}

func drawOrientationMarker() {
	// This asymmetric corner mark must appear at the physical top-left.
	for offset := 0; offset < 5; offset++ {
		for position := 12; position <= 52; position++ {
			setPortraitPixel(position, 12+offset, false)
			setPortraitPixel(12+offset, position, false)
		}
	}
}

func resetCanvas() {
	frame.Fill(true)
	drawOrientationMarker()
	if err := board.Display.FullMonochrome(frame[:]); err != nil {
		fail("RENVO PAPERMONO-LITE TOUCH TRAILS CLEAR FAIL\n")
	}
}

func (trails *trails) capture() error {
	point, pressed, err := board.Touch.Read()
	if err != nil {
		return err
	}
	changed := pressed != trails.sampledPressed
	if pressed && (point.X != trails.sampledX || point.Y != trails.sampledY) {
		changed = true
	}
	if !changed {
		return nil
	}
	if pressed {
		trails.sampledX = point.X
		trails.sampledY = point.Y
	}
	event := touchEvent{x: trails.sampledX, y: trails.sampledY, pressed: pressed}
	if trails.count < len(trails.events) {
		trails.events[trails.count] = event
		trails.count++
	} else {
		// Retain the latest position when a long recovery refresh fills the
		// queue; drawing will connect the retained samples after BUSY clears.
		trails.events[len(trails.events)-1] = event
	}
	trails.sampledPressed = pressed
	return nil
}

func (trails *trails) PollDuringRefresh() error { return trails.capture() }

func (trails *trails) applyCaptured() {
	now := board.Clock.Milliseconds()
	for index := 0; index < trails.count; index++ {
		event := trails.events[index]
		if event.pressed {
			point := ft6336g.Point{X: event.x, Y: event.y}
			if !trails.drawnPressed {
				drawBrush(point.X, point.Y)
				print("TOUCH TRAIL DOWN X=", point.X, " Y=", point.Y, "\n")
				trails.lastLog = now
			} else if point.X != trails.previous.X || point.Y != trails.previous.Y {
				drawTrail(trails.previous, point)
				if now-trails.lastLog >= logIntervalMillis {
					print("TOUCH TRAIL MOVE X=", point.X, " Y=", point.Y, "\n")
					trails.lastLog = now
				}
			}
			trails.previous = point
			trails.dirty = true
		} else if trails.drawnPressed {
			print("TOUCH TRAIL UP X=", trails.previous.X, " Y=", trails.previous.Y, "\n")
		}
		trails.drawnPressed = event.pressed
	}
	trails.count = 0
}

func presentCanvas(trails *trails) {
	started := board.Clock.Milliseconds()
	if err := board.Display.FastMonochrome(frame[:], trails); err != nil {
		fail("RENVO PAPERMONO-LITE TOUCH TRAILS PRESENT FAIL\n")
	}
	print("TOUCH TRAIL PRESENT MS=", board.Clock.Milliseconds()-started, "\n")
}

func main() {
	if board.Initialize() != nil {
		fail("RENVO PAPERMONO-LITE TOUCH TRAILS BOARD FAIL\n")
	}
	board.Clock.DelayMilliseconds(500)
	resetCanvas()
	identity, err := board.Touch.Initialize()
	if err != nil {
		fail("RENVO PAPERMONO-LITE TOUCH TRAILS INIT FAIL\n")
	}
	print("RENVO PAPERMONO-LITE TOUCH TRAILS READY CIPHER=", identity.Cipher,
		" FIRMWARE=", identity.Firmware, " VENDOR=", identity.Vendor, "\n")

	buttonA := board.ButtonA.Pressed()
	lastPresent := board.Clock.Milliseconds()
	trails := trails{lastLog: lastPresent}
	for {
		if err := trails.capture(); err != nil {
			fail("RENVO PAPERMONO-LITE TOUCH TRAILS READ FAIL\n")
		}
		trails.applyCaptured()
		now := board.Clock.Milliseconds()
		if trails.dirty && (!trails.drawnPressed || now-lastPresent >= presentIntervalMillis) {
			// The framebuffer sent below stays unchanged while the refresh poller
			// queues newer touch samples. Apply them only after RAM 2 is synced.
			trails.dirty = false
			presentCanvas(&trails)
			lastPresent = board.Clock.Milliseconds()
			trails.applyCaptured()
		}

		nextA := board.ButtonA.Pressed()
		if nextA && !buttonA {
			resetCanvas()
			trails.dirty = false
			lastPresent = board.Clock.Milliseconds()
			print("RENVO PAPERMONO-LITE TOUCH TRAILS CLEARED\n")
		}
		buttonA = nextA
		board.Clock.DelayMilliseconds(1)
	}
}
