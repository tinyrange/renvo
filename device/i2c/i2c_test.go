package i2c

import "testing"

type fakeController struct {
	configured int
	address    uint16
	writes     int
	reads      int
}

type fakeDelay struct{}

func (*fakeDelay) DelayMicroseconds(uint32) {}
func (*fakeDelay) DelayMilliseconds(uint32) {}

func (c *fakeController) Configure() error { c.configured++; return nil }
func (c *fakeController) Tx(address uint16, write, read []byte) error {
	c.address, c.writes, c.reads = address, len(write), len(read)
	return nil
}

func TestBoardPortConstructsLazyBus(t *testing.T) {
	controller := &fakeController{}
	bus := New(DefinePort(controller, &fakeDelay{}))
	if controller.configured != 0 {
		t.Fatal("constructor touched hardware")
	}
	read := make([]byte, 3)
	if err := bus.Tx(0x58, []byte{1, 2}, read); err != nil {
		t.Fatal(err)
	}
	if controller.configured != 1 || controller.address != 0x58 || controller.writes != 2 || controller.reads != 3 {
		t.Fatalf("controller = %+v", controller)
	}
}
