package main

import (
	"testing"

	"renvo.dev/device/display/ssd1677"
	"renvo.dev/device/input/ft6336g"
)

func frameWhite(controllerX, controllerY int) bool {
	index := controllerY*ssd1677.BytesPerRow + controllerX/8
	mask := byte(0x80 >> uint(controllerX&7))
	return frame[index]&mask != 0
}

func TestPortraitCornersRotateIntoControllerRAM(t *testing.T) {
	frame.Fill(true)
	setPortraitPixel(0, 0, false)
	setPortraitPixel(ft6336g.LogicalWidth-1, ft6336g.LogicalHeight-1, false)
	if frameWhite(0, ssd1677.Height-1) || frameWhite(ssd1677.Width-1, 0) {
		t.Fatal("portrait corners were not rotated into native RAM")
	}
	if !frameWhite(0, 0) || !frameWhite(ssd1677.Width-1, ssd1677.Height-1) {
		t.Fatal("portrait rotation changed the opposite native corners")
	}
}

func TestFourBoxesCoverTheirCentersAndLeaveGaps(t *testing.T) {
	for index, rectangle := range boxes {
		point := ft6336g.Point{X: rectangle.x + rectangle.width/2, Y: rectangle.y + rectangle.height/2}
		if got := boxAt(point); got != index {
			t.Fatalf("boxAt center %d = %d", index, got)
		}
	}
	for _, point := range []ft6336g.Point{{X: 0, Y: 0}, {X: 240, Y: 400}, {X: 479, Y: 799}} {
		if got := boxAt(point); got != -1 {
			t.Fatalf("boxAt gap %+v = %d", point, got)
		}
	}
}

func TestDrawBoxesAndInteriorTogglePreserveBorder(t *testing.T) {
	drawBoxes()
	rectangle := boxes[0]
	if frameWhite(rectangle.y, ssd1677.Height-1-rectangle.x) {
		t.Fatal("box border is white")
	}
	inside := interior(rectangle)
	if !frameWhite(inside.y, ssd1677.Height-1-inside.x) {
		t.Fatal("box interior did not start white")
	}
	fillPortraitRectangle(inside, false)
	if frameWhite(inside.y, ssd1677.Height-1-inside.x) {
		t.Fatal("box interior did not toggle black")
	}
	if frameWhite(rectangle.y, ssd1677.Height-1-rectangle.x) {
		t.Fatal("interior toggle changed border")
	}
}
