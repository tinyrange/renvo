// usb_midi qualifies Audio/MIDI descriptors and bulk event data.
package main

import (
	board "renvo.dev/device/board/m5sticks3"
	"renvo.dev/device/usb"
	"renvo.dev/device/usb/midi"
)

func main() {
	function := midi.New()
	device, err := usb.New(board.USB, usb.Config{VendorID: 0x1209, ProductID: 0x3477,
		Manufacturer: "Renvo", Product: "Renvo MIDI", Functions: []usb.Function{function}})
	if err != nil || device.Start() != nil {
		for {
		}
	}
	sent := false
	for {
		device.Poll()
		if !sent && function.WriteEvent([]byte{0x09, 0x90, 60, 127}) == nil {
			sent = true
		}
	}
}
