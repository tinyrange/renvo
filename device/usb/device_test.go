package usb

import "testing"

type fakeFunction struct{ configured bool }

func (*fakeFunction) Bind(b *DescriptorBuilder) error {
	i := b.Interface()
	return b.InterfaceDescriptor(i, 0, 0, 0xff, 0, 0, 0)
}
func (*fakeFunction) Attach(EndpointIO)                     {}
func (*fakeFunction) Control(*Setup, []byte) ([]byte, bool) { return nil, false }
func (*fakeFunction) ControlOut(*Setup, []byte) bool        { return false }
func (*fakeFunction) BOSDescriptor() []byte                 { return nil }
func (f *fakeFunction) Configured(value bool)               { f.configured = value }
func (*fakeFunction) Handle(Event)                          {}

type sentPacket struct {
	ep   uint8
	data []byte
}
type fakeUSBController struct {
	events  []Event
	sent    []sentPacket
	address uint8
}

func (*fakeUSBController) Start() error                      { return nil }
func (*fakeUSBController) Connect() error                    { return nil }
func (*fakeUSBController) Disconnect()                       {}
func (*fakeUSBController) OpenEndpoint(EndpointConfig) error { return nil }
func (*fakeUSBController) Receive(uint8, []byte) error       { return nil }
func (*fakeUSBController) Stall(uint8, Direction)            {}
func (c *fakeUSBController) SetAddress(address uint8)        { c.address = address }
func (c *fakeUSBController) Poll(event *Event) bool {
	if len(c.events) == 0 {
		return false
	}
	*event = c.events[0]
	c.events = c.events[1:]
	return true
}
func (c *fakeUSBController) Send(endpoint uint8, data []byte) error {
	copyData := append([]byte(nil), data...)
	c.sent = append(c.sent, sentPacket{ep: endpoint, data: copyData})
	return nil
}

func setupEvent(values [8]byte) Event { return Event{Kind: EventSetup, Setup: values} }

func TestEnumerationAndDelayedAddress(t *testing.T) {
	controller := &fakeUSBController{events: []Event{
		setupEvent([8]byte{0x80, 6, 0, 1, 0, 0, 18, 0}),
		{Kind: EventInComplete, Endpoint: 0},
		setupEvent([8]byte{0, 5, 7, 0, 0, 0, 0, 0}),
		{Kind: EventInComplete, Endpoint: 0},
	}}
	function := &fakeFunction{}
	device, err := New(DefinePort(controller), Config{VendorID: 0x1209, ProductID: 1, Functions: []Function{function}})
	if err != nil {
		t.Fatal(err)
	}
	device.Poll()
	if len(controller.sent) != 2 || len(controller.sent[0].data) != 18 {
		t.Fatalf("sent = %#v", controller.sent)
	}
	if controller.address != 7 {
		t.Fatalf("address = %d", controller.address)
	}
}

func TestConfigurationDescriptorAndSetConfiguration(t *testing.T) {
	controller := &fakeUSBController{events: []Event{
		setupEvent([8]byte{0x80, 6, 0, 2, 0, 0, 255, 0}),
		{Kind: EventInComplete, Endpoint: 0},
		setupEvent([8]byte{0, 9, 1, 0, 0, 0, 0, 0}),
		{Kind: EventInComplete, Endpoint: 0},
	}}
	function := &fakeFunction{}
	device, err := New(DefinePort(controller), Config{VendorID: 0x1209, ProductID: 1, Functions: []Function{function}})
	if err != nil {
		t.Fatal(err)
	}
	device.Poll()
	if got := controller.sent[0].data; len(got) != 18 || got[1] != 2 || got[4] != 1 {
		t.Fatalf("configuration = %v", got)
	}
	if !function.configured {
		t.Fatal("function was not configured")
	}
}
