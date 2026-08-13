package gpio

import "testing"

type fakePin struct {
	configured int
	config     Config
	level      bool
}

func (p *fakePin) Configure(config Config) error {
	p.configured++
	p.config = config
	return nil
}

func (p *fakePin) Set(level bool) { p.level = level }
func (p *fakePin) Get() bool      { return p.level }

func TestLEDLazyInitializationAndPolarity(t *testing.T) {
	pin := &fakePin{}
	led := NewLED(pin, true)
	led.Set(true)
	if pin.configured != 1 || pin.config.Direction != Output || pin.level {
		t.Fatalf("active-low LED on = configured:%d config:%+v level:%v", pin.configured, pin.config, pin.level)
	}
	led.Set(false)
	if pin.configured != 1 || !pin.level {
		t.Fatalf("second operation = configured:%d level:%v", pin.configured, pin.level)
	}
}

func TestButtonLazyInitializationAndPolarity(t *testing.T) {
	pin := &fakePin{level: true}
	button := NewButton(pin, PullUp, true)
	if button.Pressed() {
		t.Fatal("released active-low button reported pressed")
	}
	pin.level = false
	if !button.Pressed() {
		t.Fatal("active-low button did not report pressed")
	}
	if pin.configured != 1 || pin.config.Direction != Input || pin.config.Pull != PullUp {
		t.Fatalf("button config = count:%d config:%+v", pin.configured, pin.config)
	}
}
