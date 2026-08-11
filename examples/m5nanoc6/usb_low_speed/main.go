// usb_low_speed qualifies the portable software SIE, low-speed HID
// enumeration, and the ESP32-C6 raw PHY transmit path.
package main

import (
	"renvo.dev/device/esp32c6"
	"renvo.dev/device/usb"
	"renvo.dev/device/usb/hid"
	"renvo.dev/device/usb/lowspeed"
)

var proofStates [128]byte
var proofCount int

func hostToken(sie *lowspeed.SIE, pid, address, endpoint byte) {
	value := uint16(address) | uint16(endpoint)<<7
	sie.HandlePacket(pid, []byte{byte(value), byte(value >> 8)})
}

func takeReply(sie *lowspeed.SIE, want byte, output []byte) (int, bool) {
	pid, count, ok := sie.TakeReply(output)
	return count, ok && pid == want
}

func control(sie *lowspeed.SIE, device *usb.Device, address byte, setup [8]byte) (int, bool) {
	hostToken(sie, lowspeed.PIDSetup, address, 0)
	sie.HandlePacket(lowspeed.PIDData0, setup[:])
	var packet [8]byte
	if _, ok := takeReply(sie, lowspeed.PIDAck, packet[:]); !ok {
		return 0, false
	}
	device.Poll()
	if setup[0]&0x80 != 0 {
		requested := int(setup[6]) | int(setup[7])<<8
		total := 0
		for total < requested {
			hostToken(sie, lowspeed.PIDIn, address, 0)
			pid, count, ok := sie.TakeReply(packet[:])
			if !ok || pid != lowspeed.PIDData0 && pid != lowspeed.PIDData1 {
				return 0, false
			}
			total += count
			sie.HandlePacket(lowspeed.PIDAck, nil)
			device.Poll()
			if count < 8 {
				break
			}
		}
		hostToken(sie, lowspeed.PIDOut, address, 0)
		sie.HandlePacket(lowspeed.PIDData1, nil)
		if _, ok := takeReply(sie, lowspeed.PIDAck, packet[:]); !ok {
			return 0, false
		}
		device.Poll()
		return total, true
	}
	hostToken(sie, lowspeed.PIDIn, address, 0)
	count, ok := takeReply(sie, lowspeed.PIDData1, packet[:])
	if !ok || count != 0 {
		return 0, false
	}
	sie.HandlePacket(lowspeed.PIDAck, nil)
	device.Poll()
	return 0, true
}

func transmitProof() {
	payload := [8]byte{'C', '6', ' ', 'P', 'A', 'S', 'S', '\n'}
	proofCount = lowspeed.EncodeData(proofStates[:], lowspeed.PIDData0, payload[:])
	if proofCount == 0 {
		for {
		}
	}
	phy := esp32c6.NewUSBPHY(0)
	esp32c6.ArmUSBRecovery()
	phy.Takeover()
	phy.Transmit(proofStates[:proofCount])
	phy.Release()
	esp32c6.CompleteUSBRecovery()
}

func main() {
	esp32c6.OpenUSBRecoveryWindow()
	reportDescriptor := []byte{0x06, 0x00, 0xff, 0x09, 1, 0xa1, 1, 0xc0}
	function := hid.NewLowSpeed(reportDescriptor)
	sie := &lowspeed.SIE{}
	device, err := usb.New(sie.Port(), usb.Config{
		VendorID: 0x1209, ProductID: 0x00c6, ControlPacketSize: 8,
		Manufacturer: "Renvo", Product: "C6 software USB", Functions: []usb.Function{function},
	})
	if err != nil || device.Start() != nil {
		for {
		}
	}
	sie.BusReset()
	device.Poll()
	length, ok := control(sie, device, 0, [8]byte{0x80, 6, 0, 1, 0, 0, 18, 0})
	if !ok || length != 18 {
		for {
		}
	}
	control(sie, device, 0, [8]byte{0, 5, 7, 0, 0, 0, 0, 0})
	length, ok = control(sie, device, 7, [8]byte{0x80, 6, 0, 2, 0, 0, 255, 0})
	if !ok || length != 41 {
		for {
		}
	}
	control(sie, device, 7, [8]byte{0, 9, 1, 0, 0, 0, 0, 0})
	expected := [8]byte{'C', '6', ' ', 'P', 'A', 'S', 'S', '\n'}
	if !device.Configured() || function.SendReport(expected[:]) != nil {
		for {
		}
	}
	hostToken(sie, lowspeed.PIDIn, 7, 1)
	var report [8]byte
	pid, count, ok := sie.TakeReply(report[:])
	if !ok || pid != lowspeed.PIDData0 || count != 8 {
		for {
		}
	}
	for index := range expected {
		if report[index] != expected[index] {
			for {
			}
		}
	}

	transmitProof()
	for {
	}
}
