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
	polls := 0
	for {
		device.Poll()
		polls++
		if !written && serial.Write([]byte("PASS\n")) == nil {
			board.USBDownload.Complete()
			written = true
		}
		if !written && polls >= 500000 {
			board.USBDownload.Return()
			for {
				print("USBTRACE " + traceHex(board.USBDownload.Trace(0)) + " " + traceHex(board.USBDownload.Trace(1)) + " " + traceHex(board.USBDownload.Trace(2)) + " " + traceHex(board.USBDownload.Trace(3)) + "\n")
			}
		}
	}
}

func traceHex(value uint32) string {
	digits := "0123456789abcdef"
	result := make([]byte, 8)
	for index := 7; index >= 0; index-- {
		result[index] = digits[value&15]
		value >>= 4
	}
	return string(result)
}
