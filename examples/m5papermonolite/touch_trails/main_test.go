package main

import (
	"testing"

	"renvo.dev/device/display/ssd1677"
	"renvo.dev/device/input/ft6336g"
)

func portraitWhite(x, y int) bool {
	controllerX := y
	controllerY := ssd1677.Height - 1 - x
	index := controllerY*ssd1677.BytesPerRow + controllerX/8
	mask := byte(0x80 >> uint(controllerX&7))
	return frame[index]&mask != 0
}

func TestDrawTrailConnectsEndpoints(t *testing.T) {
	frame.Fill(true)
	drawTrail(ft6336g.Point{X: 40, Y: 50}, ft6336g.Point{X: 80, Y: 110})
	for _, point := range []ft6336g.Point{{X: 40, Y: 50}, {X: 60, Y: 80}, {X: 80, Y: 110}} {
		if portraitWhite(point.X, point.Y) {
			t.Fatalf("trail omitted point %+v", point)
		}
	}
	if !portraitWhite(200, 200) {
		t.Fatal("trail changed an unrelated pixel")
	}
}

func TestOrientationMarkerOccupiesOnlyTopLeft(t *testing.T) {
	frame.Fill(true)
	drawOrientationMarker()
	if portraitWhite(12, 12) || portraitWhite(52, 12) || portraitWhite(12, 52) {
		t.Fatal("top-left orientation marker is incomplete")
	}
	if !portraitWhite(ft6336g.LogicalWidth-13, 12) || !portraitWhite(12, ft6336g.LogicalHeight-13) {
		t.Fatal("orientation marker was mirrored to another corner")
	}
}
