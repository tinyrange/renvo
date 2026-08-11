// usb_cdc is the smallest native USB-OTG qualification firmware for the
// M5StickS3. It enumerates as CDC ACM and publishes a bounded PASS packet.
package main

import (
	board "renvo.dev/device/board/m5sticks3"
	"renvo.dev/device/usb"
	"renvo.dev/device/usb/cdcacm"
)

func main() {
	serial := cdcacm.New()
	device, err := usb.New(board.USB, usb.Config{
		VendorID:     0x1209,
		ProductID:    0x3470,
		DeviceBCD:    0x0100,
		Manufacturer: "Renvo",
		Product:      "Renvo CDC",
		Serial:       "S3-EMU-1",
		Functions:    []usb.Function{serial},
	})
	if err != nil || device.Start() != nil {
		for {
		}
	}
	written := false
	for {
		device.Poll()
		if !written && serial.Write([]byte("PASS\n")) == nil {
			written = true
		}
	}
}
