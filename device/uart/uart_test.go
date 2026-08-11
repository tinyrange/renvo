package uart

import (
	"testing"

	"renvo.dev/device/gpio"
)

type testPin struct{}

func (*testPin) Configure(gpio.Config) error { return nil }
func (*testPin) Set(bool)                    {}
func (*testPin) Get() bool                   { return false }

type testController struct{ configured int }

func (c *testController) Configure(gpio.Pin, gpio.Pin, uint32) error {
	c.configured++
	return nil
}
func (*testController) Write(data []byte) (int, error) { return len(data), nil }
func (*testController) Read(data []byte) (int, error) {
	data[0] = 7
	return 1, nil
}

func TestNewConfiguresLazilyOnce(t *testing.T) {
	controller, pin := &testController{}, &testPin{}
	device := New(DefinePort(controller, pin, pin, 115200))
	if count, err := device.Write([]byte{1, 2}); err != nil || count != 2 {
		t.Fatalf("write = %d, %v", count, err)
	}
	data := make([]byte, 1)
	if count, err := device.Read(data); err != nil || count != 1 || data[0] != 7 {
		t.Fatalf("read = %d, %v, %v", count, err, data)
	}
	if controller.configured != 1 {
		t.Fatalf("configured = %d", controller.configured)
	}
}
