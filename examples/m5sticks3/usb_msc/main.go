// usb_msc qualifies Mass Storage BOT with a host-originated SCSI INQUIRY.
package main

import (
	board "renvo.dev/device/board/m5sticks3"
	"renvo.dev/device/usb"
	"renvo.dev/device/usb/msc"
)

type disk struct{}

func (*disk) BlockCount() uint32 { return 2 }
func (*disk) ReadBlock(_ uint32, data []byte) error {
	for index := range data {
		data[index] = 0
	}
	return nil
}
func (*disk) WriteBlock(uint32, []byte) error { return nil }

func main() {
	function := msc.New(&disk{})
	device, err := usb.New(board.USB, usb.Config{VendorID: 0x1209, ProductID: 0x347c,
		Manufacturer: "Renvo", Product: "Renvo MSC", Functions: []usb.Function{function}})
	if err != nil || device.Start() != nil {
		for {
		}
	}
	for {
		device.Poll()
	}
}
