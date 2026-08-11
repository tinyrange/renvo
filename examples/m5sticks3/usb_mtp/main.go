// usb_mtp qualifies the still-image interface, status control, bulk container,
// and interrupt event endpoint.
package main

import (
	board "renvo.dev/device/board/m5sticks3"
	"renvo.dev/device/usb"
	"renvo.dev/device/usb/mtp"
)

func main() {
	function := mtp.New()
	device, err := usb.New(board.USB, usb.Config{VendorID: 0x1209, ProductID: 0x347a,
		Manufacturer: "Renvo", Product: "Renvo MTP", Functions: []usb.Function{function}})
	if err != nil || device.Start() != nil {
		for {
		}
	}
	sent := false
	for {
		device.Poll()
		if !sent && function.WriteContainer(mtp.Container{Type: mtp.ContainerResponse,
			Code: 0x2001, Transaction: 7}, []byte("MTP PASS\n")) == nil {
			_ = function.SendEvent([]byte{12, 0, 0, 0, 4, 0, 2, 0x40, 7, 0, 0, 0})
			sent = true
		}
	}
}
