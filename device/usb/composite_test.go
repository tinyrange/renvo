package usb_test

import (
	"testing"

	"renvo.dev/device/usb"
	"renvo.dev/device/usb/adb"
	"renvo.dev/device/usb/audio"
	"renvo.dev/device/usb/cdcacm"
	"renvo.dev/device/usb/cdcethernet"
	"renvo.dev/device/usb/hid"
	"renvo.dev/device/usb/midi"
	"renvo.dev/device/usb/msc"
	"renvo.dev/device/usb/mtp"
	"renvo.dev/device/usb/vendor"
)

type testPacket struct {
	endpoint uint8
	data     []byte
}

type testController struct {
	events    []usb.Event
	opened    []usb.EndpointConfig
	sent      []testPacket
	receiving []uint8
}

func (*testController) Start() error   { return nil }
func (*testController) Connect() error { return nil }
func (*testController) Disconnect()    {}
func (c *testController) Poll(event *usb.Event) bool {
	if len(c.events) == 0 {
		return false
	}
	*event = c.events[0]
	c.events = c.events[1:]
	return true
}
func (c *testController) OpenEndpoint(config usb.EndpointConfig) error {
	c.opened = append(c.opened, config)
	return nil
}
func (c *testController) Send(endpoint uint8, data []byte) error {
	c.sent = append(c.sent, testPacket{endpoint: endpoint, data: append([]byte(nil), data...)})
	return nil
}
func (c *testController) Receive(endpoint uint8, _ []byte) error {
	c.receiving = append(c.receiving, endpoint)
	return nil
}
func (*testController) Stall(uint8, usb.Direction) {}
func (*testController) SetAddress(uint8)           {}

func setup(bytes [8]byte) usb.Event { return usb.Event{Kind: usb.EventSetup, Setup: bytes} }

type testDisk struct{ blocks [2][512]byte }

func (*testDisk) BlockCount() uint32 { return 2 }
func (d *testDisk) ReadBlock(block uint32, data []byte) error {
	copy(data, d.blocks[block][:])
	return nil
}
func (d *testDisk) WriteBlock(block uint32, data []byte) error {
	copy(d.blocks[block][:], data)
	return nil
}

func TestEveryClassBuildsADeviceProfile(t *testing.T) {
	profiles := []struct {
		name     string
		function usb.Function
	}{
		{"cdc-acm", cdcacm.New()},
		{"hid", hid.New([]byte{5, 1, 9, 6, 0xa1, 1, 0xc0})},
		{"webusb", vendor.NewWebUSB(0x22, "renvo.dev")},
		{"mass-storage", msc.New(&testDisk{})},
		{"midi", midi.New()},
		{"cdc-ethernet", cdcethernet.New()},
		{"audio", audio.New(48000)},
		{"mtp", mtp.New()},
		{"adb", adb.New()},
	}
	for _, profile := range profiles {
		t.Run(profile.name, func(t *testing.T) {
			controller := &testController{}
			device, err := usb.New(usb.DefinePort(controller), usb.Config{
				VendorID: 0x1209, ProductID: 0x3472, Functions: []usb.Function{profile.function},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := device.Start(); err != nil {
				t.Fatal(err)
			}
			for _, endpoint := range controller.opened {
				if endpoint.Number > 6 {
					t.Fatalf("endpoint %d exceeds ESP32-S3 budget", endpoint.Number)
				}
			}
		})
	}
}

func TestDeveloperCompositeFitsS3EndpointBudget(t *testing.T) {
	controller := &testController{}
	functions := []usb.Function{
		cdcacm.New(),
		hid.New([]byte{5, 1, 9, 6, 0xa1, 1, 0xc0}),
		vendor.NewWebUSB(0x22, "renvo.dev"),
	}
	device, err := usb.New(usb.DefinePort(controller), usb.Config{
		VendorID: 0x1209, ProductID: 0x3470, Functions: functions,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Start(); err != nil {
		t.Fatal(err)
	}
	maximum := uint8(0)
	for _, endpoint := range controller.opened {
		if endpoint.Number > maximum {
			maximum = endpoint.Number
		}
	}
	if maximum != 4 {
		t.Fatalf("maximum endpoint number = %d, want 4", maximum)
	}
	controller.events = []usb.Event{
		setup([8]byte{0x80, 6, 0, 2, 0, 0, 255, 0}),
		{Kind: usb.EventInComplete},
		{Kind: usb.EventInComplete},
		{Kind: usb.EventInComplete},
	}
	device.Poll()
	var descriptor []byte
	for _, packet := range controller.sent {
		if packet.endpoint == 0 {
			descriptor = append(descriptor, packet.data...)
		}
	}
	if len(descriptor) < 9 || descriptor[1] != 2 || descriptor[4] != 4 {
		t.Fatalf("configuration descriptor = %v", descriptor)
	}
}

func TestWebUSBBOSHasConsistentLengths(t *testing.T) {
	function := vendor.NewWebUSB(0x22, "renvo.dev")
	bos := function.BOSDescriptor()
	if len(bos) != 29 || int(bos[2])|int(bos[3])<<8 != len(bos) {
		t.Fatalf("BOS length bytes = %v, actual = %d", bos[:5], len(bos))
	}
	if bos[5] != 24 || bos[6] != 16 || bos[7] != 5 {
		t.Fatalf("platform capability header = %v", bos[5:9])
	}
}

func TestCDCLineCodingControlOutDataStage(t *testing.T) {
	controller := &testController{}
	serial := cdcacm.New()
	device, err := usb.New(usb.DefinePort(controller), usb.Config{
		VendorID: 0x1209, ProductID: 0x3471, Functions: []usb.Function{serial},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Start(); err != nil {
		t.Fatal(err)
	}
	wanted := []byte{0x00, 0x4b, 0x00, 0x00, 0, 0, 8}
	controller.events = []usb.Event{
		setup([8]byte{0, 9, 1, 0, 0, 0, 0, 0}),
		{Kind: usb.EventInComplete},
		setup([8]byte{0x21, 0x20, 0, 0, 0, 0, 7, 0}),
		{Kind: usb.EventOut, Endpoint: 0, Data: wanted},
		setup([8]byte{0xa1, 0x21, 0, 0, 0, 0, 7, 0}),
	}
	device.Poll()
	got := controller.sent[len(controller.sent)-1].data
	if string(got) != string(wanted) {
		t.Fatalf("line coding = %v, want %v", got, wanted)
	}
}

func TestMassStorageInquiryAndCommandStatus(t *testing.T) {
	controller := &testController{}
	storage := msc.New(&testDisk{})
	device, err := usb.New(usb.DefinePort(controller), usb.Config{
		VendorID: 0x1209, ProductID: 0x3473, Functions: []usb.Function{storage},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Start(); err != nil {
		t.Fatal(err)
	}
	var out, in uint8
	for _, endpoint := range controller.opened {
		if endpoint.Direction == usb.Out {
			out = endpoint.Number
		} else {
			in = endpoint.Number
		}
	}
	cbw := make([]byte, 31)
	copy(cbw, []byte{'U', 'S', 'B', 'C'})
	cbw[4] = 7
	cbw[8] = 36
	cbw[12] = 0x80
	cbw[14] = 6
	cbw[15] = 0x12
	cbw[19] = 36
	controller.events = []usb.Event{
		setup([8]byte{0, 9, 1, 0, 0, 0, 0, 0}),
		{Kind: usb.EventInComplete},
		{Kind: usb.EventOut, Endpoint: out, Data: cbw},
		{Kind: usb.EventInComplete, Endpoint: in},
	}
	device.Poll()
	var transfers [][]byte
	for _, packet := range controller.sent {
		if packet.endpoint == in {
			transfers = append(transfers, packet.data)
		}
	}
	if len(transfers) != 2 || len(transfers[0]) != 36 || string(transfers[0][8:16]) != "RENVO   " {
		t.Fatalf("inquiry transfers = %v", transfers)
	}
	if len(transfers[1]) != 13 || string(transfers[1][:4]) != "USBS" || transfers[1][4] != 7 || transfers[1][12] != 0 {
		t.Fatalf("CSW = %v", transfers[1])
	}
}

func TestAudioAlternateSettingControlsStreaming(t *testing.T) {
	controller := &testController{}
	function := audio.New(48000)
	device, err := usb.New(usb.DefinePort(controller), usb.Config{
		VendorID: 0x1209, ProductID: 0x3474, Functions: []usb.Function{function},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := device.Start(); err != nil {
		t.Fatal(err)
	}
	controller.events = []usb.Event{
		setup([8]byte{0, 9, 1, 0, 0, 0, 0, 0}),
		{Kind: usb.EventInComplete},
		setup([8]byte{0x01, 11, 1, 0, 1, 0, 0, 0}),
		setup([8]byte{0x81, 10, 0, 0, 1, 0, 1, 0}),
	}
	device.Poll()
	if got := controller.sent[len(controller.sent)-1].data; len(got) != 1 || got[0] != 1 {
		t.Fatalf("GET_INTERFACE = %v", got)
	}
	armed := false
	for _, endpoint := range controller.receiving {
		if endpoint != 0 {
			armed = true
		}
	}
	if !armed {
		t.Fatal("audio OUT endpoint was not armed by alternate setting 1")
	}
}
