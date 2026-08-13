package lowspeed

import (
	"testing"

	"renvo.dev/device/usb"
	"renvo.dev/device/usb/hid"
)

func hostToken(s *SIE, pid, address, endpoint byte) {
	value := uint16(address) | uint16(endpoint)<<7
	s.HandlePacket(pid, []byte{byte(value), byte(value >> 8)})
}

func takeReply(t *testing.T, s *SIE, wantPID byte) []byte {
	t.Helper()
	var data [8]byte
	pid, count, ok := s.TakeReply(data[:])
	if !ok || pid != wantPID {
		t.Fatalf("reply pid=%02x count=%d ok=%v, want %02x", pid, count, ok, wantPID)
	}
	return append([]byte(nil), data[:count]...)
}

func controlTransfer(t *testing.T, s *SIE, device *usb.Device, address byte, setup [8]byte) []byte {
	t.Helper()
	hostToken(s, PIDSetup, address, 0)
	s.HandlePacket(PIDData0, setup[:])
	takeReply(t, s, PIDAck)
	device.Poll()

	if setup[0]&0x80 != 0 {
		requested := int(setup[6]) | int(setup[7])<<8
		var result []byte
		for len(result) < requested {
			hostToken(s, PIDIn, address, 0)
			var packet [8]byte
			pid, count, ok := s.TakeReply(packet[:])
			if !ok || pid != PIDData0 && pid != PIDData1 {
				t.Fatalf("control IN pid=%02x count=%d ok=%v", pid, count, ok)
			}
			result = append(result, packet[:count]...)
			s.HandlePacket(PIDAck, nil)
			device.Poll()
			if count < 8 {
				break
			}
		}
		hostToken(s, PIDOut, address, 0)
		s.HandlePacket(PIDData1, nil)
		takeReply(t, s, PIDAck)
		device.Poll()
		return result
	}

	hostToken(s, PIDIn, address, 0)
	if reply := takeReply(t, s, PIDData1); len(reply) != 0 {
		t.Fatalf("status IN returned %d bytes", len(reply))
	}
	s.HandlePacket(PIDAck, nil)
	device.Poll()
	return nil
}

func TestSIEEnumeratesLowSpeedHIDAndTransfersReport(t *testing.T) {
	reportDescriptor := []byte{0x06, 0x00, 0xff, 0x09, 1, 0xa1, 1, 0xc0}
	function := hid.NewLowSpeed(reportDescriptor)
	sie := &SIE{}
	device, err := usb.New(sie.Port(), usb.Config{
		VendorID: 0x1209, ProductID: 0x00c6, ControlPacketSize: 8,
		Manufacturer: "Renvo", Product: "C6 software USB", Functions: []usb.Function{function},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Start(); err != nil {
		t.Fatal(err)
	}
	sie.BusReset()
	device.Poll()

	deviceDescriptor := controlTransfer(t, sie, device, 0, [8]byte{0x80, 6, 0, 1, 0, 0, 18, 0})
	if len(deviceDescriptor) != 18 || deviceDescriptor[7] != 8 {
		t.Fatalf("device descriptor = %v", deviceDescriptor)
	}
	controlTransfer(t, sie, device, 0, [8]byte{0, 5, 7, 0, 0, 0, 0, 0})
	configuration := controlTransfer(t, sie, device, 7, [8]byte{0x80, 6, 0, 2, 0, 0, 255, 0})
	if len(configuration) != 41 || configuration[31] != 8 || configuration[38] != 8 {
		t.Fatalf("low-speed HID configuration = %v", configuration)
	}
	controlTransfer(t, sie, device, 7, [8]byte{0, 9, 1, 0, 0, 0, 0, 0})
	if !device.Configured() {
		t.Fatal("device was not configured")
	}

	if err := function.SendReport([]byte("C6 PASS\n")); err != nil {
		t.Fatal(err)
	}
	hostToken(sie, PIDIn, 7, 1)
	if got := string(takeReply(t, sie, PIDData0)); got != "C6 PASS\n" {
		t.Fatalf("report = %q", got)
	}
	sie.HandlePacket(PIDAck, nil)
	device.Poll()
}
