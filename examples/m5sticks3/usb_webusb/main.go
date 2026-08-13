// usb_webusb qualifies BOS, landing-page control, and vendor bulk data.
package main

import (
	board "renvo.dev/device/board/m5sticks3"
	"renvo.dev/device/usb"
	"renvo.dev/device/usb/vendor"
)

func main() {
	function := vendor.NewWebUSB(0x22, "renvo.dev")
	device, err := usb.New(board.USB, usb.Config{VendorID: 0x1209, ProductID: 0x3479,
		Manufacturer: "Renvo", Product: "Renvo WebUSB", Functions: []usb.Function{function}})
	if err != nil || device.Start() != nil {
		for {
		}
	}
	sent := false
	for {
		device.Poll()
		if !sent && function.Write([]byte("WEBUSB PASS\n")) == nil {
			sent = true
		}
	}
}
