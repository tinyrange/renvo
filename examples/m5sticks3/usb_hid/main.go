// usb_hid qualifies HID descriptors, control requests, and interrupt IN data.
package main

import (
	board "renvo.dev/device/board/m5sticks3"
	"renvo.dev/device/usb"
	"renvo.dev/device/usb/hid"
)

func main() {
	// One vendor-defined eight-byte input report and one output report.
	report := []byte{0x06, 0x00, 0xff, 0x09, 1, 0xa1, 1, 0x15, 0, 0x26, 0xff, 0,
		0x75, 8, 0x95, 8, 0x09, 1, 0x81, 2, 0x95, 8, 0x09, 1, 0x91, 2, 0xc0}
	function := hid.New(report)
	device, err := usb.New(board.USB, usb.Config{VendorID: 0x1209, ProductID: 0x3475,
		Manufacturer: "Renvo", Product: "Renvo HID", Functions: []usb.Function{function}})
	if err != nil || device.Start() != nil {
		for {
		}
	}
	sent := false
	for {
		device.Poll()
		if !sent && function.SendReport([]byte("HID PASS\n")) == nil {
			sent = true
		}
	}
}
