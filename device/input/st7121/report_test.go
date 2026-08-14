package st7121

import "testing"

func TestDecodeReport(t *testing.T) {
	page := make([]byte, ReportSize(2))
	page[0] = 0x08
	page[4] = 0x80 | 0x02
	page[5] = 0x34
	page[6] = 0x01
	page[7] = 0x23
	page[8] = 7
	page[9] = 41
	page[11] = 0x80 | 0x03
	page[12] = 0xff
	page[13] = 0x01
	page[14] = 0x20

	var points [MaximumContacts]Point
	advanced, count, ok := DecodeReport(page, 2, NativeWidth, NativeHeight, points[:])
	if !ok || advanced != 0x08 || count != 1 {
		t.Fatalf("decoded report = advanced %#x count %d ok %v", advanced, count, ok)
	}
	if points[0] != (Point{ID: 0, X: 0x234, Y: 0x123, Strength: 7, Intensity: 41}) {
		t.Fatalf("decoded point = %#v", points[0])
	}
}

func TestDecodeReportRejectsShortPage(t *testing.T) {
	if _, _, ok := DecodeReport(make([]byte, ReportSize(1)-1), 1, NativeWidth, NativeHeight, nil); ok {
		t.Fatal("short report accepted")
	}
}
