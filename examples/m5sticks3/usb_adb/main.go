// usb_adb qualifies the ADB interface and framed bulk transport.
package main

import (
	board "renvo.dev/device/board/m5sticks3"
	"renvo.dev/device/usb"
	"renvo.dev/device/usb/adb"
)

func main() {
	function := adb.New()
	device, err := usb.New(board.USB, usb.Config{VendorID: 0x1209, ProductID: 0x347b,
		Manufacturer: "Renvo", Product: "Renvo ADB", Functions: []usb.Function{function}})
	if err != nil || device.Start() != nil {
		for {
		}
	}
	sent := false
	for {
		device.Poll()
		if !sent && function.WriteMessage(adb.Message{Command: adb.CommandConnect,
			Argument0: 0x01000001, Argument1: 4096}, []byte("ADB PASS\n")) == nil {
			sent = true
		}
	}
}
