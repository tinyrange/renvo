package main

import (
	"renvo.dev/device/board"
	"renvo.dev/device/display/ssd1677"
	"renvo.dev/device/input/ft6336g"
)

const borderThickness = 8

type box struct {
	x, y, width, height int
}

var boxes = [4]box{
	{x: 24, y: 24, width: 208, height: 368},
	{x: 248, y: 24, width: 208, height: 368},
	{x: 24, y: 408, width: 208, height: 368},
	{x: 248, y: 408, width: 208, height: 368},
}

var frame ssd1677.Monochrome
var black [4]bool

func fail(message string) {
	print(message)
	_ = board.Display.Shutdown()
	for {
	}
}

// The panel's visible portrait coordinates are rotated into SSD1677 native
// RAM: logical (x,y) becomes controller (y,479-x).
func setPortraitPixel(x, y int, white bool) {
	frame.Set(y, ssd1677.Height-1-x, white)
}

func fillPortraitRectangle(rectangle box, white bool) {
	for y := rectangle.y; y < rectangle.y+rectangle.height; y++ {
		for x := rectangle.x; x < rectangle.x+rectangle.width; x++ {
			setPortraitPixel(x, y, white)
		}
	}
}

func interior(rectangle box) box {
	return box{
		x:      rectangle.x + borderThickness,
		y:      rectangle.y + borderThickness,
		width:  rectangle.width - borderThickness*2,
		height: rectangle.height - borderThickness*2,
	}
}

func drawBoxes() {
	frame.Fill(true)
	for _, rectangle := range boxes {
		fillPortraitRectangle(rectangle, false)
		fillPortraitRectangle(interior(rectangle), true)
	}
}

func boxAt(point ft6336g.Point) int {
	for index, rectangle := range boxes {
		if point.X >= rectangle.x && point.X < rectangle.x+rectangle.width &&
			point.Y >= rectangle.y && point.Y < rectangle.y+rectangle.height {
			return index
		}
	}
	return -1
}

func toggle(index int, point ft6336g.Point) {
	black[index] = !black[index]
	fillPortraitRectangle(interior(boxes[index]), !black[index])
	if err := board.Display.PartialMonochrome(frame[:]); err != nil {
		fail("RENVO PAPERMONO-LITE TOUCH REFRESH FAIL\n")
	}
	print("TOUCH BOX ", index+1)
	if black[index] {
		print(" BLACK")
	} else {
		print(" WHITE")
	}
	print(" X=", point.X, " Y=", point.Y, "\n")
}

func reportButton(name string, pressed bool) {
	print("BUTTON ", name)
	if pressed {
		print(" DOWN\n")
	} else {
		print(" UP\n")
	}
}

func main() {
	if board.Initialize() != nil {
		fail("RENVO PAPERMONO-LITE TOUCH BOARD FAIL\n")
	}
	board.Clock.DelayMilliseconds(500)
	drawBoxes()
	if err := board.Display.FullMonochrome(frame[:]); err != nil {
		fail("RENVO PAPERMONO-LITE TOUCH BASELINE FAIL\n")
	}
	identity, err := board.Touch.Initialize()
	if err != nil {
		fail("RENVO PAPERMONO-LITE TOUCH INIT FAIL\n")
	}
	print("RENVO PAPERMONO-LITE TOUCH READY CIPHER=", identity.Cipher,
		" FIRMWARE=", identity.Firmware, " VENDOR=", identity.Vendor, "\n")

	pressed := false
	buttonA := board.ButtonA.Pressed()
	buttonB := board.ButtonB.Pressed()
	for {
		point, nextPressed, err := board.Touch.Read()
		if err != nil {
			fail("RENVO PAPERMONO-LITE TOUCH READ FAIL\n")
		}
		if nextPressed && !pressed {
			index := boxAt(point)
			if index >= 0 {
				toggle(index, point)
			} else {
				print("TOUCH OUTSIDE X=", point.X, " Y=", point.Y, "\n")
			}
		}
		pressed = nextPressed

		nextA := board.ButtonA.Pressed()
		nextB := board.ButtonB.Pressed()
		if nextA != buttonA {
			buttonA = nextA
			reportButton("A", buttonA)
		}
		if nextB != buttonB {
			buttonB = nextB
			reportButton("B", buttonB)
		}
		board.Clock.DelayMilliseconds(20)
	}
}
