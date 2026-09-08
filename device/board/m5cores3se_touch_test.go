//go:build m5cores3se

package board

import "testing"

func TestCoreTouchReport(t *testing.T) {
	var points [2]TouchPoint
	report := []byte{2, 0x80, 120, 0x10, 80, 0, 0, 0x81, 63, 0x20, 239, 0, 0}
	n, ok := decodeCoreTouch(report, points[:])
	if !ok || n != 2 || points[0] != (TouchPoint{X: 120, Y: 80, ID: 1}) || points[1] != (TouchPoint{X: 319, Y: 239, ID: 2}) {
		t.Fatalf("decode = %v %d %v", points, n, ok)
	}
	report[1] = 0x40 // released contact
	n, ok = decodeCoreTouch(report, points[:])
	if !ok || n != 1 || points[0].ID != 2 {
		t.Fatal("released contact retained")
	}
	report[10] = 250 // below physical display
	n, ok = decodeCoreTouch(report, points[:])
	if !ok || n != 0 {
		t.Fatal("out-of-display contact retained")
	}
	report[0] = 3
	if _, ok = decodeCoreTouch(report, points[:]); ok {
		t.Fatal("invalid count accepted")
	}
	if _, ok = decodeCoreTouch(report[:2], points[:]); ok {
		t.Fatal("short report accepted")
	}
}

func TestCoreNativeRGB565Packing(t *testing.T) {
	pixels := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0xf8, 0xe0, 7, 0x1f, 0, 0xff, 0xff}
	var buffer dmaBuffer
	used := packRGB5652xRow(pixels, 8, 1, 4, 1, &buffer)
	if used != 24 {
		t.Fatal(used)
	}
	want := []uint32{0xe007e007, 0x1f001f00, 0xffffffff}
	for i := 0; i < 3; i++ {
		if buffer[i] != want[i] || buffer[i+3] != want[i] {
			t.Fatalf("native pixel %d: %x", i, buffer[:6])
		}
	}
}
