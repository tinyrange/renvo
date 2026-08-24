package uart

import (
	"testing"
)

type fakeController struct {
	configurations int
	baud           uint32
	writes         [][]byte
	err            error
}

func (c *fakeController) Configure(baud uint32) error {
	c.configurations++
	c.baud = baud
	return c.err
}

func (c *fakeController) Write(data []byte) (int, error) {
	c.writes = append(c.writes, append([]byte(nil), data...))
	return len(data), c.err
}

func TestPortConfiguresLazilyAndOnlyOnce(t *testing.T) {
	controller := &fakeController{}
	transmitter := New(DefinePort(controller), 31250)
	if controller.configurations != 0 {
		t.Fatal("New touched the controller")
	}
	if n, err := transmitter.Write([]byte{0x90, 60, 100}); err != nil || n != 3 {
		t.Fatalf("first Write = %d, %v", n, err)
	}
	if n, err := transmitter.Write([]byte{0x80, 60, 0}); err != nil || n != 3 {
		t.Fatalf("second Write = %d, %v", n, err)
	}
	if controller.configurations != 1 || controller.baud != 31250 || len(controller.writes) != 2 {
		t.Fatalf("controller = %#v", controller)
	}
}
