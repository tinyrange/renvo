// usb_cdc_ethernet qualifies CDC ECM controls and bulk network data.
package main

import (
	board "renvo.dev/device/board/m5sticks3"
	"renvo.dev/device/usb"
	"renvo.dev/device/usb/cdcethernet"
)

func main() {
	function := cdcethernet.New()
	device, err := usb.New(board.USB, usb.Config{VendorID: 0x1209, ProductID: 0x3476,
		Manufacturer: "Renvo", Product: "Renvo ECM", Functions: []usb.Function{function}})
	if err != nil || device.Start() != nil {
		for {
		}
	}
	sent := false
	for {
		device.Poll()
		if !sent && function.WriteFrame([]byte("ECM PASS\n")) == nil {
			sent = true
		}
	}
}
