package spi

import (
	"testing"

	"renvo.dev/device/gpio"
)

type testPin struct{}

func (*testPin) Configure(gpio.Config) error { return nil }
func (*testPin) Set(bool)                    {}
func (*testPin) Get() bool                   { return false }

type testController struct{ configured int }

func (c *testController) Configure(gpio.Pin, gpio.Pin, gpio.Pin, uint32, uint8) error {
	c.configured++
	return nil
}
func (*testController) Tx(write, read []byte) error {
	copy(read, write)
	return nil
}

func TestNewConfiguresLazilyOnce(t *testing.T) {
	controller, pin := &testController{}, &testPin{}
	bus := New(DefinePort(controller, pin, pin, nil, 10_000_000, 0))
	read := make([]byte, 2)
	if err := bus.Tx([]byte{1, 2}, read); err != nil {
		t.Fatal(err)
	}
	if err := bus.Tx(nil, nil); err != nil {
		t.Fatal(err)
	}
	if controller.configured != 1 || read[0] != 1 || read[1] != 2 {
		t.Fatalf("configured=%d read=%v", controller.configured, read)
	}
}
