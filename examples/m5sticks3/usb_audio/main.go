// usb_audio qualifies UAC1 alternate settings and a 192-byte isochronous IN transfer.
package main

import (
	board "renvo.dev/device/board/m5sticks3"
	"renvo.dev/device/usb"
	"renvo.dev/device/usb/audio"
)

func main() {
	function := audio.New(48000)
	device, err := usb.New(board.USB, usb.Config{VendorID: 0x1209, ProductID: 0x3478,
		Manufacturer: "Renvo", Product: "Renvo Audio", Functions: []usb.Function{function}})
	if err != nil || device.Start() != nil {
		for {
		}
	}
	packet := make([]byte, 192)
	copy(packet, []byte("AUDIO PASS\n"))
	for index := len("AUDIO PASS\n"); index < len(packet); index++ {
		packet[index] = 0x5a
	}
	packet[len(packet)-1] = 0xa5
	sent := false
	for {
		device.Poll()
		if !sent && function.WriteSamples(packet) == nil {
			sent = true
		}
	}
}
